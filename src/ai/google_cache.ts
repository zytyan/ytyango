import type { AiRequest, ProviderState } from "./types.ts";
import { cacheContent } from "./messages.ts";

const baseURL = "https://generativelanguage.googleapis.com/v1beta";

/** Only cache lifecycle endpoints lack a generateText equivalent. No background API calls. */
export async function explicitCache(
  request: AiRequest,
  key: string,
  fetcher: typeof fetch,
): Promise<{ state: ProviderState; tailStart: number } | undefined> {
  if (!request.explicitCache || request.provider !== "gemini" || request.historyRebuildLossy) {
    return undefined;
  }
  const now = Math.floor(Date.now() / 1000);
  const stableCount = request.messages.length - request.newMessages.length;
  if (stableCount <= 0) return undefined;
  const previousEnd = request.messages.findIndex((m) =>
    m.messageId === request.state.geminiCacheEndMessageId
  );
  const reusable = request.state.geminiCacheName && previousEnd >= 0 && previousEnd < stableCount &&
    (request.state.geminiCacheExpireTime ?? 0) > now + 300 &&
    request.messages[0]?.messageId === request.state.geminiCacheStartMessageId &&
    request.messages.length - previousEnd - 1 < 50;
  const stable = request.messages.slice(0, reusable ? previousEnd + 1 : stableCount);
  const contents = stable.map(cacheContent);
  if (contents.some((content) => !content)) return undefined;
  const systemInstruction = { role: "model", parts: [{ text: request.systemPrompt }] };
  const tools = [
    { googleSearch: {} },
    ...(request.allowCodeExecution ? [{ codeExecution: {} }] : []),
  ];
  const context = { model: `models/${request.model}`, contents, systemInstruction, tools };
  const fingerprint = Array.from(
    new Uint8Array(
      await crypto.subtle.digest(
        "SHA-256",
        new TextEncoder().encode(JSON.stringify(context)),
      ),
    ),
    (byte) => byte.toString(16).padStart(2, "0"),
  ).join("");
  if (reusable && fingerprint === request.state.geminiCacheFingerprint) {
    return { state: request.state, tailStart: previousEnd + 1 };
  }
  // Rebuild against all stable messages if the previous cache's prompt/options no longer match.
  if (stable.length !== stableCount) return undefined;
  if (now - (stable.at(-1)?.sentAt ?? now) < 300) return undefined;
  const post = async (
    path: string,
    payload: unknown,
  ): Promise<Record<string, unknown> | undefined> => {
    const response = await fetcher(`${baseURL}/${path}`, {
      method: "POST",
      headers: { "x-goog-api-key": key, "content-type": "application/json" },
      body: JSON.stringify(payload),
      signal: request.signal,
    });
    if (!response.ok) {
      await response.body?.cancel();
      if ([400, 404, 501].includes(response.status)) return undefined;
      throw new Error(`Gemini cache API HTTP ${response.status}`);
    }
    return await response.json();
  };
  const counted = await post(`models/${request.model}:countTokens`, {
    generateContentRequest: context,
  });
  if (Number(counted?.totalTokens ?? 0) < 4096) return undefined;
  const created = await post("cachedContents", {
    ...context,
    displayName: `ytyan-session-${request.sessionId}`,
    ttl: "900s",
  });
  if (typeof created?.name !== "string" || typeof created.expireTime !== "string") return undefined;
  const expireTime = Math.floor(Date.parse(created.expireTime) / 1000);
  if (!Number.isFinite(expireTime) || expireTime <= now) return undefined;
  return {
    tailStart: stableCount,
    state: {
      geminiCacheName: created.name,
      geminiCacheExpireTime: expireTime,
      geminiCacheStartMessageId: stable[0]?.messageId,
      geminiCacheEndMessageId: stable.at(-1)?.messageId,
      geminiCacheTokenCount: Number(counted?.totalTokens ?? 0),
      geminiCacheFingerprint: fingerprint,
    },
  };
}
