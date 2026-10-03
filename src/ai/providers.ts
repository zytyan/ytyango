import { APICallError, generateText } from "ai";
import { createGoogle } from "@ai-sdk/google";
import { createDeepSeek } from "@ai-sdk/deepseek";
import type { AppConfig } from "../config.ts";
import type { AiRequest, AiResult, ProviderState } from "./types.ts";
import { MODEL_GEMINI_37_FLASH, MODEL_GEMINI_FLASH } from "./types.ts";
import {
  type ApiMode,
  modelMessages,
  replayLegacy,
  savedPayload,
  SDK_PAYLOAD_FORMAT,
} from "./messages.ts";
import { explicitCache } from "./google_cache.ts";

const encoder = new TextEncoder();
const bytes = (value: unknown) => encoder.encode(JSON.stringify(value));
const compatibleError = (error: unknown) =>
  APICallError.isInstance(error) && [400, 404, 501].includes(error.statusCode ?? 0);

/** One entry point, direct SDK model instances; application state stays in the Run transaction. */
export function createAiGenerator(config: () => AppConfig, fetcher: typeof fetch = fetch) {
  return async (input: AiRequest): Promise<AiResult> => {
    input.signal?.throwIfAborted();
    const request: AiRequest = {
      ...input,
      signal: input.signal
        ? AbortSignal.any([input.signal, AbortSignal.timeout(180_000)])
        : AbortSignal.timeout(180_000),
    };
    const settings = config();
    if (!["gemini", "sub2api", "deepseek"].includes(request.provider)) {
      throw new Error(`unsupported AI provider: ${request.provider}`);
    }
    const key = request.provider === "deepseek"
      ? settings.deepSeekKey
      : request.provider === "sub2api"
      ? settings.sub2apiKey
      : settings.geminiKey;
    if (!key) throw new Error(`${request.provider} API Key 未配置`);
    const last = request.messages.at(-1);
    if (
      request.provider === "deepseek" && !last?.text &&
      (last?.messageType === "video" || last?.messageType === "animation")
    ) {
      throw new Error("当前模型不处理视频");
    }

    const call = async (
      mode: ApiMode,
      source = request.messages,
      state: ProviderState = {},
      previousInteractionId?: string,
    ): Promise<AiResult> => {
      let raw = new Uint8Array();
      const capture: typeof fetch = async (url, init) => {
        const body = JSON.parse(String(init?.body)) as Record<string, unknown>;
        replayLegacy(body, source, mode, request.historyRebuildLossy);
        const response = await fetcher(url, { ...init, body: JSON.stringify(body) });
        // Preserve original bytes, including fields the SDK does not understand.
        raw = new Uint8Array(await response.clone().arrayBuffer());
        return response;
      };
      const google = createGoogle({
        apiKey: key,
        fetch: capture,
        ...(request.provider === "sub2api" ? { baseURL: settings.sub2apiBaseUrl } : {}),
        ...(mode === "interactions" ? { headers: { "Api-Revision": "2026-05-20" } } : {}),
      });
      const model = mode === "deepseek"
        ? createDeepSeek({ apiKey: key, baseURL: settings.deepSeekBaseUrl, fetch: capture })(
          request.model,
        )
        : mode === "interactions"
        ? google.interactions(request.model)
        : google(
          request.provider === "sub2api" && request.model === MODEL_GEMINI_37_FLASH
            ? "gemini-3.7-flash-tiered"
            : request.model,
        );
      const result = await generateText({
        model,
        messages: modelMessages(
          source,
          request.provider,
          request.model,
          mode,
          request.historyRebuildLossy,
        ),
        system: state.geminiCacheName ? undefined : request.systemPrompt,
        // A single attempt: no stacked retry loops, no automatic re-execution of paid work.
        maxRetries: 0,
        abortSignal: request.signal,
        include: { responseBody: true },
        tools: mode === "deepseek" || state.geminiCacheName ? undefined : {
          google_search: google.tools.googleSearch({}),
          ...(request.allowCodeExecution ? { code_execution: google.tools.codeExecution({}) } : {}),
        },
        providerOptions: mode === "deepseek" ? { deepseek: { thinking: { type: "enabled" } } } : {
          google: mode === "interactions"
            ? {
              store: true,
              previousInteractionId,
              thinkingSummaries: "auto",
              ...(request.model === MODEL_GEMINI_FLASH ? { thinkingLevel: "low" } : {}),
            }
            : {
              cachedContent: state.geminiCacheName,
              thinkingConfig: {
                includeThoughts: true,
                ...(request.model === MODEL_GEMINI_FLASH ? { thinkingLevel: "low" } : {}),
              },
            },
        },
      });
      const native = JSON.parse(new TextDecoder().decode(raw)) as Record<string, unknown>;
      const interactionId = result.providerMetadata?.google?.interactionId;
      if (mode === "interactions" && (typeof interactionId !== "string" || !interactionId)) {
        throw new Error("Gemini Interactions API did not return an interaction id");
      }
      const candidates = native.candidates as { content?: Record<string, unknown> }[] | undefined;
      const choices = native.choices as { message?: Record<string, unknown> }[] | undefined;
      const nativeAssistant = mode === "deepseek"
        ? choices?.[0]?.message
        : mode === "interactions"
        ? (native.steps as Record<string, unknown>[] | undefined)?.filter((step) =>
          step.type !== "user_input"
        )
        : candidates?.[0]?.content;
      const signature = result.response.messages.flatMap((m) =>
        m.role === "assistant" && Array.isArray(m.content) ? m.content : []
      ).map((part) =>
        "providerOptions" in part
          ? part.providerOptions?.google?.thoughtSignature ??
            part.providerOptions?.google?.signature
          : undefined
      )
        .find((value) => typeof value === "string");
      const blockReason = (native.promptFeedback as { blockReason?: string } | undefined)
        ?.blockReason;
      return {
        displayText: result.text,
        assistantPayload: bytes({
          provider: request.provider,
          mode,
          messages: result.response.messages,
          native: nativeAssistant,
        }),
        assistantPayloadFormat: SDK_PAYLOAD_FORMAT,
        usage: {
          inputTokens: result.usage.inputTokens ?? 0,
          cachedInputTokens: result.usage.inputTokenDetails.cacheReadTokens ?? 0,
          outputTokens: result.usage.outputTokens ?? 0,
          thinkingTokens: result.usage.outputTokenDetails.reasoningTokens ?? 0,
        },
        raw,
        thoughtSignature: typeof signature === "string" ? signature : undefined,
        feedback: blockReason ??
          (result.finishReason === "content-filter" ? "content-filter" : undefined),
        state: mode === "interactions"
          ? {
            geminiInteractionId: String(interactionId),
            windowStartMessageId: request.messages[0]?.messageId,
          }
          : state,
        inputMessageCount: request.messages.length,
        inputFirstMessageId: request.messages[0]?.messageId,
        inputLastMessageId: request.messages.at(-1)?.messageId,
      };
    };

    if (request.provider === "deepseek") return await call("deepseek");
    if (request.provider === "sub2api") return await call("generateContent");
    const cache = await explicitCache(request, key, fetcher);
    if (cache) {
      try {
        return await call("generateContent", request.messages.slice(cache.tailStart), cache.state);
      } catch (error) {
        if (!compatibleError(error)) throw error;
      }
    }
    const linked = !request.historyRebuildLossy &&
        request.state.windowStartMessageId === request.messages[0]?.messageId
      ? request.state.geminiInteractionId
      : undefined;
    // New cache-enabled sessions use native generation so their signed payload is cacheable.
    // Existing remote Interactions chains retain that protocol and are never cached lossily.
    const interactionHistory = request.messages.some((message) =>
      message.assistantPayloadFormat === "gemini-interaction-steps-v1" ||
      (message.assistantPayloadFormat === SDK_PAYLOAD_FORMAT &&
        (savedPayload(message) as { mode?: string }).mode === "interactions")
    );
    if (request.explicitCache && !linked && !interactionHistory) {
      return await call("generateContent");
    }
    try {
      try {
        return await call(
          "interactions",
          linked ? request.newMessages : request.messages,
          {},
          linked,
        );
      } catch (error) {
        if (!linked || !compatibleError(error)) throw error;
        return await call("interactions");
      }
    } catch (error) {
      if (!compatibleError(error)) throw error;
      return await call("generateContent");
    }
  };
}
