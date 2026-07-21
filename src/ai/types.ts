export const MODEL_GEMINI_FLASH = "gemini-3-flash-preview";
export const MODEL_GEMINI_FLASH_LITE = "gemini-3.1-flash-lite";
export const MODEL_DEEPSEEK_FLASH = "deepseek-v4-flash";

export type AiProviderName = "gemini" | "deepseek";
export type AiRole = "user" | "model" | "system";

export interface AiMedia {
  readonly kind: string;
  readonly mimeType: string;
  readonly bytes: Uint8Array;
}

export interface AiMessage {
  readonly chatId: number;
  readonly messageId: number;
  readonly sentAt: number;
  readonly userId: number;
  readonly username: string;
  readonly atableUsername?: string;
  readonly role: AiRole;
  readonly messageType: string;
  readonly text?: string;
  readonly replyToMessageId?: number;
  readonly quotePart?: string;
  readonly media: readonly AiMedia[];
  readonly assistantPayload?: Uint8Array;
  readonly assistantPayloadFormat?: string;
}

export interface AiUsage {
  readonly inputTokens: number;
  readonly cachedInputTokens: number;
  readonly outputTokens: number;
  readonly thinkingTokens: number;
}

export interface ProviderState {
  readonly geminiInteractionId?: string;
  readonly windowStartMessageId?: number;
  readonly geminiCacheName?: string;
  readonly geminiCacheExpireTime?: number;
  readonly geminiCacheStartMessageId?: number;
  readonly geminiCacheEndMessageId?: number;
  readonly geminiCacheTokenCount?: number;
  readonly geminiCacheFingerprint?: string;
}

export interface AiRequest {
  readonly sessionId: number;
  readonly provider: AiProviderName;
  readonly model: string;
  readonly systemPrompt: string;
  readonly messages: readonly AiMessage[];
  readonly newMessages: readonly AiMessage[];
  readonly historyRebuildLossy: boolean;
  readonly allowCodeExecution: boolean;
  readonly explicitCache: boolean;
  readonly state: ProviderState;
  readonly signal?: AbortSignal;
}

export interface AiResult {
  readonly displayText: string;
  readonly assistantPayload: Uint8Array;
  readonly assistantPayloadFormat: string;
  readonly usage: AiUsage;
  readonly raw: Uint8Array;
  readonly thoughtSignature?: string;
  readonly feedback?: string;
  readonly state: ProviderState;
  readonly inputMessageCount: number;
  readonly inputFirstMessageId?: number;
  readonly inputLastMessageId?: number;
}

export interface AiProvider {
  readonly name: AiProviderName;
  generate(request: AiRequest): Promise<AiResult>;
}

export interface AiSession {
  readonly id: number;
  readonly chatId: number;
  readonly topicId?: number;
  readonly provider: AiProviderName;
  readonly model: string;
  readonly historyRebuildLossy: boolean;
}

export function providerForModel(model: string): AiProviderName {
  return model === MODEL_DEEPSEEK_FLASH ? "deepseek" : "gemini";
}

export const MODEL_OPTIONS = Object.freeze([
  { model: MODEL_GEMINI_FLASH, label: "Gemini 3 Flash Preview", provider: "gemini" as const },
  { model: MODEL_GEMINI_FLASH_LITE, label: "Gemini 3.1 Flash-Lite", provider: "gemini" as const },
  { model: MODEL_DEEPSEEK_FLASH, label: "DeepSeek V4 Flash", provider: "deepseek" as const },
]);
