import type { AppConfig } from "../config.ts";
import type {
  AiMessage,
  AiProvider,
  AiRequest,
  AiResult,
  AiUsage,
  ProviderState,
} from "./types.ts";
import { MODEL_GEMINI_FLASH } from "./types.ts";

const encoder = new TextEncoder();
const decoder = new TextDecoder();

type Fetcher = typeof fetch;

function jsonBytes(value: unknown): Uint8Array {
  return encoder.encode(JSON.stringify(value));
}

function base64(bytes: Uint8Array): string {
  let binary = "";
  for (let offset = 0; offset < bytes.length; offset += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
  }
  return btoa(binary);
}

function compactText(message: AiMessage): string {
  const date = new Date(message.sentAt * 1000);
  const timestamp = new Intl.DateTimeFormat("sv-SE", {
    timeZone: "Asia/Shanghai",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(date).replace(" ", " ");
  return `[ ${message.username} ${timestamp} ]\n${message.text ?? ""}`;
}

function generationContent(message: AiMessage): Record<string, unknown> {
  const parts: Record<string, unknown>[] = [{
    text: message.role === "user" ? compactText(message) : message.text ?? "",
  }];
  for (const media of message.media) {
    parts.push({ inlineData: { mimeType: media.mimeType, data: base64(media.bytes) } });
  }
  if (
    message.role === "model" && message.assistantPayloadFormat === "gemini-content-v1" &&
    message.assistantPayload
  ) {
    try {
      return JSON.parse(decoder.decode(message.assistantPayload));
    } catch {
      // Fall back to the portable text stored in ai_messages.
    }
  }
  return { role: message.role === "model" ? "model" : "user", parts };
}

function interactionSteps(message: AiMessage, lossy: boolean): unknown[] {
  if (message.role === "model") {
    if (
      !lossy && message.assistantPayloadFormat === "gemini-interaction-steps-v1" &&
      message.assistantPayload
    ) {
      try {
        const parsed = JSON.parse(decoder.decode(message.assistantPayload));
        if (Array.isArray(parsed) && parsed.length > 0) return parsed;
      } catch {
        // Corrupt historical payloads degrade to saved display text.
      }
    }
    return message.text
      ? [{ type: "model_output", content: [{ type: "text", text: message.text }] }]
      : [];
  }
  if (lossy && message.media.length > 0 && !message.text) return [];
  const content: Record<string, unknown>[] = [{ type: "text", text: compactText(message) }];
  if (!lossy) {
    for (const media of message.media) {
      if (media.mimeType.startsWith("image/") || media.mimeType.startsWith("video/")) {
        content.push({
          type: media.mimeType.startsWith("image/") ? "image" : "video",
          mime_type: media.mimeType,
          data: base64(media.bytes),
        });
      }
    }
  }
  return [{ type: "user_input", content }];
}

function usage(
  inputTokens = 0,
  cachedInputTokens = 0,
  outputTokens = 0,
  thinkingTokens = 0,
): AiUsage {
  return { inputTokens, cachedInputTokens, outputTokens, thinkingTokens };
}

async function responseBody(
  response: Response,
): Promise<{ raw: Uint8Array; value: Record<string, unknown> }> {
  const raw = new Uint8Array(await response.arrayBuffer());
  let value: Record<string, unknown> = {};
  try {
    value = JSON.parse(decoder.decode(raw));
  } catch {
    // HTTP status and a bounded text body are reported by the caller.
  }
  return { raw, value };
}

function apiError(service: string, response: Response, body: Record<string, unknown>): Error {
  const nested = body.error as Record<string, unknown> | undefined;
  const message = typeof nested?.message === "string" ? nested.message : response.statusText;
  return new Error(`${service} HTTP ${response.status}: ${message}`);
}

export class DeepSeekProvider implements AiProvider {
  readonly name = "deepseek" as const;

  constructor(readonly config: () => AppConfig, readonly fetcher: Fetcher = fetch) {}

  async generate(request: AiRequest): Promise<AiResult> {
    const config = this.config();
    if (!config.deepSeekKey) throw new Error("DeepSeek API Key 未配置");
    const last = request.messages.at(-1);
    if (
      last?.role === "user" && !last.text &&
      (last.messageType === "video" || last.messageType === "animation")
    ) {
      throw new Error("当前模型不处理视频");
    }
    const converted = request.messages.flatMap((message) => {
      if (message.role === "system") return [];
      if (message.role === "user") {
        let portable = compactText(message);
        if (message.messageType === "image") portable += "\n[图片]";
        if (message.messageType === "sticker" && !message.text) portable += "\n[贴纸]";
        return portable.trim() || message.media.length === 0
          ? [{ role: "user", content: portable }]
          : [];
      }
      if (
        !request.historyRebuildLossy && message.assistantPayloadFormat === "deepseek-message-v1" &&
        message.assistantPayload
      ) {
        try {
          return [JSON.parse(decoder.decode(message.assistantPayload))];
        } catch {
          // Fall through to portable text.
        }
      }
      return message.text ? [{ role: "assistant", content: message.text }] : [];
    });
    if (!converted.some((message) => message.role === "user")) {
      throw new Error("当前模型不处理视频");
    }
    const payload = {
      model: request.model,
      messages: [{ role: "system", content: request.systemPrompt }, ...converted],
      thinking: { type: "enabled" },
    };
    let lastError: unknown;
    for (let attempt = 0; attempt < 3; attempt++) {
      try {
        const response = await this.fetcher(`${config.deepSeekBaseUrl}/chat/completions`, {
          method: "POST",
          headers: {
            authorization: `Bearer ${config.deepSeekKey}`,
            "content-type": "application/json",
          },
          body: JSON.stringify(payload),
          signal: request.signal,
        });
        const body = await responseBody(response);
        if (!response.ok) throw apiError("DeepSeek API", response, body.value);
        const choices = body.value.choices as
          | Array<{ message: Record<string, unknown> }>
          | undefined;
        if (!choices?.[0]?.message) throw new Error("DeepSeek API 没有返回候选结果");
        const message = choices[0].message;
        const metrics = body.value.usage as Record<string, unknown> | undefined;
        const details = metrics?.completion_tokens_details as Record<string, unknown> | undefined;
        return {
          displayText: typeof message.content === "string" ? message.content : "",
          assistantPayload: jsonBytes(message),
          assistantPayloadFormat: "deepseek-message-v1",
          usage: usage(
            Number(metrics?.prompt_tokens ?? 0),
            Number(metrics?.prompt_cache_hit_tokens ?? 0),
            Number(metrics?.completion_tokens ?? 0),
            Number(details?.reasoning_tokens ?? 0),
          ),
          raw: body.raw,
          state: {},
          inputMessageCount: request.messages.length,
          inputFirstMessageId: request.messages.at(0)?.messageId,
          inputLastMessageId: request.messages.at(-1)?.messageId,
        };
      } catch (error) {
        lastError = error;
        if (attempt < 2) await new Promise((resolve) => setTimeout(resolve, 1000 * 2 ** attempt));
      }
    }
    throw lastError;
  }
}

export class GeminiProvider implements AiProvider {
  readonly name = "gemini" as const;
  readonly #baseUrl = "https://generativelanguage.googleapis.com/v1beta";

  constructor(readonly config: () => AppConfig, readonly fetcher: Fetcher = fetch) {}

  async generate(request: AiRequest): Promise<AiResult> {
    const key = this.config().geminiKey;
    if (!key) throw new Error("Gemini API Key 未配置");
    const cached = request.explicitCache ? await this.#tryExplicitCache(request, key) : undefined;
    if (cached) return cached;
    try {
      return await this.#interactions(request, key);
    } catch (error) {
      const status = (error as { status?: number }).status;
      if (status !== 400 && status !== 404 && status !== 501) throw error;
      return await this.#generateContent(request, key, request.messages, request.systemPrompt, {});
    }
  }

  async #interactions(request: AiRequest, key: string): Promise<AiResult> {
    const rebuilding = !request.state.geminiInteractionId || request.historyRebuildLossy;
    const source = rebuilding ? request.messages : request.newMessages;
    const input = source.flatMap((message) =>
      interactionSteps(
        message,
        request.historyRebuildLossy && !request.newMessages.includes(message),
      )
    );
    const payload: Record<string, unknown> = {
      model: request.model,
      input,
      system_instruction: request.systemPrompt,
      tools: [
        { type: "google_search" },
        ...(request.allowCodeExecution ? [{ type: "code_execution" }] : []),
      ],
      store: true,
      generation_config: {
        thinking_summaries: "auto",
        ...(request.model === MODEL_GEMINI_FLASH ? { thinking_level: "low" } : {}),
      },
    };
    if (!rebuilding) payload.previous_interaction_id = request.state.geminiInteractionId;
    const call = async (): Promise<
      { response: Response; raw: Uint8Array; value: Record<string, unknown> }
    > => {
      const response = await this.fetcher(`${this.#baseUrl}/interactions`, {
        method: "POST",
        headers: {
          "content-type": "application/json",
          "x-goog-api-key": key,
          "Api-Revision": "2026-05-20",
        },
        body: JSON.stringify(payload),
        signal: request.signal,
      });
      const body = await responseBody(response);
      return { response, ...body };
    };
    let body = await call();
    if (
      !body.response.ok && payload.previous_interaction_id &&
      (body.response.status === 400 || body.response.status === 404)
    ) {
      payload.input = request.messages.flatMap((message) =>
        interactionSteps(message, request.historyRebuildLossy)
      );
      delete payload.previous_interaction_id;
      body = await call();
    }
    if (!body.response.ok) {
      const error = apiError("Gemini Interactions API", body.response, body.value) as Error & {
        status?: number;
      };
      error.status = body.response.status;
      throw error;
    }
    const interactionId = String(body.value.id ?? "");
    if (!interactionId) throw new Error("Gemini Interactions API did not return an interaction id");
    const steps = Array.isArray(body.value.steps)
      ? body.value.steps as Array<Record<string, unknown>>
      : [];
    const modelSteps = steps.filter((step) => step.type !== "user_input");
    let displayText = "";
    for (const step of steps) {
      if (step.type !== "model_output") continue;
      const content = Array.isArray(step.content)
        ? step.content as Array<Record<string, unknown>>
        : [];
      const text = content.filter((part) => part.type === "text").map((part) =>
        String(part.text ?? "")
      ).join("");
      if (text) displayText = text;
    }
    const metrics = body.value.usage as Record<string, unknown> | undefined;
    return {
      displayText,
      assistantPayload: jsonBytes(modelSteps),
      assistantPayloadFormat: "gemini-interaction-steps-v1",
      usage: usage(
        Number(metrics?.total_input_tokens ?? 0),
        Number(metrics?.total_cached_tokens ?? 0),
        Number(metrics?.total_output_tokens ?? 0) + Number(metrics?.total_thought_tokens ?? 0),
        Number(metrics?.total_thought_tokens ?? 0),
      ),
      raw: body.raw,
      feedback: body.value.status && body.value.status !== "completed"
        ? String(body.value.status)
        : undefined,
      state: {
        geminiInteractionId: interactionId,
        windowStartMessageId: request.messages.at(0)?.messageId,
      },
      inputMessageCount: request.messages.length,
      inputFirstMessageId: request.messages.at(0)?.messageId,
      inputLastMessageId: request.messages.at(-1)?.messageId,
    };
  }

  async #tryExplicitCache(request: AiRequest, key: string): Promise<AiResult | undefined> {
    const stableCount = request.messages.length - request.newMessages.length;
    if (stableCount <= 0) return undefined;
    const stable = request.messages.slice(0, stableCount);
    const tail = request.messages.slice(stableCount);
    const now = Math.floor(Date.now() / 1000);
    let state = request.state;
    const valid = state.geminiCacheName && state.geminiCacheExpireTime &&
      state.geminiCacheExpireTime > now + 300 &&
      state.geminiCacheStartMessageId === stable.at(0)?.messageId &&
      request.messages.some((message) => message.messageId === state.geminiCacheEndMessageId) &&
      request.messages.length - request.messages.findIndex((message) =>
              message.messageId === state.geminiCacheEndMessageId
            ) -
            1 < 50;
    if (!valid) {
      const lastStable = stable.at(-1);
      if (!lastStable || now - lastStable.sentAt < 300) return undefined;
      const contents = stable.map(generationContent);
      const counted = await this.#post(
        `${this.#baseUrl}/models/${request.model}:countTokens`,
        key,
        { contents },
        request.signal,
      );
      if (!counted) return undefined;
      if (Number(counted.value.totalTokens ?? 0) < 4096) return undefined;
      const created = await this.#post(
        `${this.#baseUrl}/cachedContents`,
        key,
        {
          model: `models/${request.model}`,
          displayName: `ytyan-session-${request.sessionId}`,
          ttl: "900s",
          systemInstruction: { role: "model", parts: [{ text: request.systemPrompt }] },
          contents,
          tools: [
            { googleSearch: {} },
            ...(request.allowCodeExecution ? [{ codeExecution: {} }] : []),
          ],
        },
        request.signal,
        true,
      );
      if (!created) return undefined;
      const name = String(created.value.name ?? "");
      if (!name) return undefined;
      state = {
        geminiCacheName: name,
        geminiCacheExpireTime: Math.floor(
          new Date(String(created.value.expireTime ?? Date.now() + 900_000)).getTime() / 1000,
        ),
        geminiCacheStartMessageId: stable[0]?.messageId,
        geminiCacheEndMessageId: stable.at(-1)?.messageId,
        geminiCacheTokenCount: Number(
          (created.value.usageMetadata as Record<string, unknown> | undefined)?.totalTokenCount ??
            counted.value.totalTokens ?? 0,
        ),
      };
    }
    if (!state.geminiCacheName) return undefined;
    return await this.#generateContent(request, key, tail, undefined, state);
  }

  async #generateContent(
    request: AiRequest,
    key: string,
    messages: readonly AiMessage[],
    systemPrompt: string | undefined,
    state: ProviderState,
  ): Promise<AiResult> {
    const config: Record<string, unknown> = {
      tools: [{ googleSearch: {} }, ...(request.allowCodeExecution ? [{ codeExecution: {} }] : [])],
      thinkingConfig: {
        includeThoughts: true,
        ...(request.model === MODEL_GEMINI_FLASH ? { thinkingLevel: "LOW" } : {}),
      },
    };
    if (systemPrompt) config.systemInstruction = { role: "model", parts: [{ text: systemPrompt }] };
    if (state.geminiCacheName) {
      config.cachedContent = state.geminiCacheName;
      delete config.tools;
      delete config.systemInstruction;
    }
    const body = await this.#post(`${this.#baseUrl}/models/${request.model}:generateContent`, key, {
      contents: messages.map(generationContent),
      ...config,
    }, request.signal);
    if (!body) throw new Error("Gemini API returned no response");
    const candidates = body.value.candidates as Array<Record<string, unknown>> | undefined;
    const content = candidates?.[0]?.content as Record<string, unknown> | undefined;
    const parts = Array.isArray(content?.parts)
      ? content.parts as Array<Record<string, unknown>>
      : [];
    const displayText = parts.filter((part) => !part.thought).map((part) => String(part.text ?? ""))
      .join("");
    const thought = parts.find((part) => part.thoughtSignature);
    const metrics = body.value.usageMetadata as Record<string, unknown> | undefined;
    return {
      displayText,
      assistantPayload: jsonBytes(content ?? { role: "model", parts: [] }),
      assistantPayloadFormat: "gemini-content-v1",
      usage: usage(
        Number(metrics?.promptTokenCount ?? 0),
        Number(metrics?.cachedContentTokenCount ?? 0),
        Number(metrics?.candidatesTokenCount ?? 0) + Number(metrics?.thoughtsTokenCount ?? 0),
        Number(metrics?.thoughtsTokenCount ?? 0),
      ),
      raw: body.raw,
      thoughtSignature: typeof thought?.thoughtSignature === "string"
        ? thought.thoughtSignature
        : undefined,
      feedback: (body.value.promptFeedback as Record<string, unknown> | undefined)?.blockReason as
        | string
        | undefined,
      state,
      inputMessageCount: request.messages.length,
      inputFirstMessageId: request.messages.at(0)?.messageId,
      inputLastMessageId: request.messages.at(-1)?.messageId,
    };
  }

  async #post(
    url: string,
    key: string,
    payload: unknown,
    signal?: AbortSignal,
    optional = false,
  ): Promise<{ raw: Uint8Array; value: Record<string, unknown> } | undefined> {
    const response = await this.fetcher(url, {
      method: "POST",
      headers: { "content-type": "application/json", "x-goog-api-key": key },
      body: JSON.stringify(payload),
      signal,
    });
    const body = await responseBody(response);
    if (!response.ok) {
      if (optional && [400, 404, 501].includes(response.status)) return undefined;
      throw apiError("Gemini API", response, body.value);
    }
    return body;
  }
}
