import type { AppConfig } from "../config.ts";
import { KeyedMutex } from "../shared/concurrency.ts";
import type { Clock, MediaStore, TelegramPort } from "../shared/types.ts";
import { systemClock } from "../shared/types.ts";
import type { AiRepository, PendingAiMessage, RunRow, StoredAiMessage } from "./repository.ts";
import type { AiMessage, AiProvider, AiResult, AiSession } from "./types.ts";

export const DEFAULT_SYSTEM_PROMPT = `这里是一个Telegram聊天 type:%CHAT_TYPE%,name:%CHAT_NAME%
你是一个Telegram机器人，name: %BOT_NAME% username: %BOT_USERNAME%
你会看到多轮消息。用户消息以 \`[ 显示名 YYYY-MM-DD HH:mm:ss ]\` 开头，时间为 Asia/Shanghai 时区；下一行是正文，随后可能包含媒体内容。
该紧凑消息头只用于说明发送者和时间，不要在模型输出中重复消息头。
应使用中文回复消息。
请对大家温柔一些。`;

export interface IncomingMedia {
  readonly fileId: string;
  readonly uniqueId?: string;
  readonly kind: "image" | "video" | "sticker" | "animation";
  readonly mimeType: string;
  readonly byteSize?: number;
  readonly duration?: number;
}

export interface IncomingAiMessage {
  readonly chatId: number;
  readonly chatName: string;
  readonly chatType: string;
  readonly topicId?: number;
  readonly messageId: number;
  readonly sentAt: number;
  readonly userId: number;
  readonly username: string;
  readonly atableUsername?: string;
  readonly text?: string;
  readonly reply?: IncomingAiMessage;
  readonly media: readonly IncomingMedia[];
}

export interface BotIdentity {
  readonly id: number;
  readonly firstName: string;
  readonly username: string;
}

export interface SessionRouting {
  readonly createNew: boolean;
  readonly ignoreCurrent: boolean;
  readonly explicitSessionId?: number;
}

export function parseSessionRouting(text: string): SessionRouting {
  if (text.includes("@new")) return { createNew: true, ignoreCurrent: false };
  if (text.includes("@last")) return { createNew: false, ignoreCurrent: true };
  const match = text.match(/@(\d+)/);
  return {
    createNew: false,
    ignoreCurrent: false,
    explicitSessionId: match ? Number(match[1]) : undefined,
  };
}

function topicKey(message: IncomingAiMessage): string {
  return `${message.chatId}:${message.topicId ?? 0}`;
}

function storedResult(run: RunRow): AiResult {
  let state = {};
  try {
    state = run.candidate_state_json ? JSON.parse(run.candidate_state_json) : {};
  } catch {
    // Invalid state does not prevent redelivery of already generated text.
  }
  return {
    displayText: run.response_text ?? "",
    assistantPayload: run.assistant_payload ?? new Uint8Array(),
    assistantPayloadFormat: run.assistant_payload_format ?? "",
    usage: {
      inputTokens: run.input_tokens ?? 0,
      outputTokens: run.output_tokens ?? 0,
      cachedInputTokens: run.cached_input_tokens ?? 0,
      thinkingTokens: 0,
    },
    raw: run.raw_payload ?? new Uint8Array(),
    thoughtSignature: run.thought_signature ?? undefined,
    state,
    inputMessageCount: run.input_message_count ?? 0,
    inputFirstMessageId: run.input_first_msg_id ?? undefined,
    inputLastMessageId: run.input_last_msg_id ?? undefined,
  };
}

export class AiService {
  readonly #current = new Map<string, number>();
  readonly #locks = new KeyedMutex();

  constructor(
    readonly repository: AiRepository,
    readonly providers: ReadonlyMap<string, AiProvider>,
    readonly telegram: TelegramPort,
    readonly mediaStore: MediaStore,
    readonly config: () => AppConfig,
    readonly botIdentity: BotIdentity,
    readonly clock: Clock = systemClock,
  ) {}

  resetCurrent(chatId: number, topicId = 0): void {
    this.#current.delete(`${chatId}:${topicId}`);
  }

  currentSessionId(chatId: number, topicId = 0): number | undefined {
    return this.#current.get(`${chatId}:${topicId}`);
  }

  sessionForMessage(chatId: number, messageId: number): AiSession | undefined {
    return this.repository.findSessionByMessage(chatId, messageId);
  }

  async handle(message: IncomingAiMessage): Promise<void> {
    if (!this.config().aiChats.includes(message.chatId)) return;
    const routing = parseSessionRouting(message.text ?? "");
    const target = this.#resolveSession(message, routing);
    await this.#locks.use(String(target.id), async () => {
      await this.#handleLocked(target, message);
    });
  }

  #resolveSession(message: IncomingAiMessage, routing: SessionRouting): AiSession {
    let target: AiSession | undefined;
    if (!routing.createNew && routing.explicitSessionId) {
      const explicit = this.repository.getSession(routing.explicitSessionId);
      if (explicit?.chatId === message.chatId) target = explicit;
    }
    if (!routing.createNew && !target && message.reply) {
      target = this.repository.findSessionByMessage(message.chatId, message.reply.messageId);
    }
    if (!routing.createNew && !target) {
      const id = this.#current.get(topicKey(message));
      if (id) target = this.repository.getSession(id);
    }
    if (!target) {
      const selection = this.repository.getSelection(message.chatId);
      target = this.repository.createSession({
        chatId: message.chatId,
        topicId: message.topicId,
        chatName: message.chatName,
        chatType: message.chatType,
        provider: selection.provider,
        model: selection.model,
        now: Math.floor(this.clock.now().getTime() / 1000),
      });
    }
    this.#current.set(topicKey(message), target.id);
    return target;
  }

  async #handleLocked(initialSession: AiSession, incoming: IncomingAiMessage): Promise<void> {
    const session = this.repository.getSession(initialSession.id) ?? initialSession;
    try {
      await this.telegram.setReaction(incoming.chatId, incoming.messageId, "👀");
    } catch {
      // Reactions are best effort.
    }
    const pending: PendingAiMessage[] = [];
    if (
      incoming.reply &&
      !this.repository.findSessionByMessage(incoming.chatId, incoming.reply.messageId)
    ) {
      pending.push(await this.#prepareMessage(incoming.reply, true, session.provider));
    }
    pending.push(await this.#prepareMessage(incoming, false, session.provider));
    const now = Math.floor(this.clock.now().getTime() / 1000);
    const existing = this.repository.getRunByRequest(
      session.id,
      incoming.chatId,
      incoming.messageId,
    );
    const run = existing ?? this.repository.beginRun(session.id, pending, now);
    if (run.status === "delivered") return;
    if (run.status === "model_failed" || run.status === "persistence_failed") {
      throw new Error(`previous AI attempt failed (${run.status})`);
    }
    let result: AiResult;
    if (run.status === "generated" || run.status === "delivery_failed") {
      result = storedResult(run);
    } else {
      try {
        const loaded = await this.#loadMessages(session.id);
        const window = loaded.length > 200 ? loaded.slice(50) : loaded;
        const provider = this.providers.get(session.provider);
        if (!provider) throw new Error(`unsupported AI provider: ${session.provider}`);
        const promptTemplate =
          this.repository.getSystemPrompt(incoming.chatId, incoming.topicId ?? 0) ??
            DEFAULT_SYSTEM_PROMPT;
        result = await provider.generate({
          sessionId: session.id,
          provider: session.provider,
          model: session.model,
          systemPrompt: this.#renderPrompt(promptTemplate, incoming),
          messages: window,
          newMessages: window.filter((item) =>
            pending.some((value) => value.message.messageId === item.messageId)
          ),
          historyRebuildLossy: session.historyRebuildLossy,
          allowCodeExecution: window.every((item) =>
            item.media.every((media) => !media.mimeType.startsWith("video/"))
          ),
          explicitCache: this.config().geminiExplicitCache,
          state: this.repository.loadProviderState(session.id),
        });
        this.repository.completeRun(run.id, result, Math.floor(this.clock.now().getTime() / 1000));
      } catch (error) {
        this.repository.failRun(
          run.id,
          "model_failed",
          error,
          Math.floor(this.clock.now().getTime() / 1000),
        );
        try {
          await this.telegram.setReaction(incoming.chatId, incoming.messageId, "😭");
        } catch {
          // Best effort.
        }
        await this.telegram.sendMessage(incoming.chatId, `error:${String(error)}`, {
          reply_parameters: { message_id: incoming.messageId },
        });
        throw error;
      }
    }
    let display = result.displayText;
    if (!display) {
      display = `模型没有返回任何信息${result.feedback ? `，原因: ${result.feedback}` : ""}`;
    }
    const options: Record<string, unknown> = {
      reply_parameters: { message_id: incoming.messageId },
    };
    if (this.repository.getSelection(incoming.chatId).showUsage) {
      options.reply_markup = {
        inline_keyboard: [[{ text: "📊 Token 用量", callback_data: "aiusage" }]],
      };
    }
    try {
      const response = await this.telegram.sendRichMessage(incoming.chatId, display, options);
      this.repository.deliverRun(
        run.id,
        {
          chatId: response.chatId,
          messageId: response.messageId,
          sentAt: Math.floor(this.clock.now().getTime() / 1000),
          userId: this.botIdentity.id,
          username: this.botIdentity.firstName,
          atableUsername: this.botIdentity.username,
          role: "model",
          messageType: "text",
          text: display,
          replyToMessageId: incoming.messageId,
          media: [],
        },
        result,
        Math.floor(this.clock.now().getTime() / 1000),
      );
    } catch (error) {
      this.repository.failRun(
        run.id,
        "delivery_failed",
        error,
        Math.floor(this.clock.now().getTime() / 1000),
      );
      throw error;
    }
  }

  async #prepareMessage(
    input: IncomingAiMessage,
    contextOnly: boolean,
    provider: string,
  ): Promise<PendingAiMessage> {
    const mediaObjects: Array<Awaited<ReturnType<MediaStore["put"]>> & { kind: string }> = [];
    for (const media of input.media) {
      if (
        provider === "deepseek" &&
        (media.kind === "video" || media.kind === "animation" || media.kind === "sticker")
      ) continue;
      if ((media.byteSize ?? 0) > 15 * 1024 * 1024 || (media.duration ?? 0) > 240) continue;
      const file = await this.telegram.getFile(media.fileId);
      const bytes = await this.telegram.downloadFile(file);
      if (bytes.byteLength > 15 * 1024 * 1024) continue;
      mediaObjects.push({ ...await this.mediaStore.put(bytes, media.mimeType), kind: media.kind });
    }
    const message: AiMessage = {
      chatId: input.chatId,
      messageId: input.messageId,
      sentAt: input.sentAt,
      userId: input.userId,
      username: input.username,
      atableUsername: input.atableUsername,
      role: "user",
      messageType: input.media[0]?.kind ?? "text",
      text: input.text,
      replyToMessageId: input.reply?.messageId,
      media: [],
    };
    return { message, mediaObjects, contextOnly };
  }

  async #loadMessages(sessionId: number): Promise<AiMessage[]> {
    const stored = this.repository.loadMessages(sessionId);
    return await Promise.all(stored.map(async (message: StoredAiMessage): Promise<AiMessage> => ({
      ...message,
      media: await Promise.all(message.mediaObjects.map(async (media) => ({
        kind: media.kind,
        mimeType: media.mimeType,
        bytes: await this.mediaStore.open(media.sha256),
      }))),
    })));
  }

  #renderPrompt(template: string, message: IncomingAiMessage): string {
    return template
      .replaceAll("%CHAT_TYPE%", message.chatType)
      .replaceAll("%CHAT_NAME%", message.chatName)
      .replaceAll("%BOT_NAME%", this.botIdentity.firstName)
      .replaceAll("%BOT_USERNAME%", this.botIdentity.username);
  }
}
