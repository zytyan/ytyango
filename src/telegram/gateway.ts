import { Bot, InputFile } from "grammy";
import type { Message } from "grammy/types";
import type { SentMessage, TelegramFile, TelegramPort, UploadSource } from "../shared/types.ts";

function sent(message: Message): SentMessage {
  return { chatId: message.chat.id, messageId: message.message_id, raw: message };
}

export class GrammyTelegramGateway implements TelegramPort {
  constructor(
    readonly bot: Bot,
    readonly token: string,
    readonly apiRoot: string,
  ) {}

  async sendMessage(
    chatId: number,
    text: string,
    options: Record<string, unknown> = {},
  ): Promise<SentMessage> {
    return sent(await this.bot.api.sendMessage(chatId, text, options));
  }

  async sendPhoto(
    chatId: number,
    photo: UploadSource,
    options: Record<string, unknown> = {},
  ): Promise<SentMessage> {
    return sent(await this.bot.api.sendPhoto(chatId, this.#input(photo), options));
  }

  async sendVideo(
    chatId: number,
    video: UploadSource,
    options: Record<string, unknown> = {},
  ): Promise<SentMessage> {
    return sent(await this.bot.api.sendVideo(chatId, this.#input(video), options));
  }

  async sendAudio(
    chatId: number,
    audio: UploadSource,
    options: Record<string, unknown> = {},
  ): Promise<SentMessage> {
    return sent(await this.bot.api.sendAudio(chatId, this.#input(audio), options));
  }

  async sendRichMessage(
    chatId: number,
    text: string,
    options: Record<string, unknown> = {},
  ): Promise<SentMessage> {
    const payload = {
      chat_id: chatId,
      rich_message: { markdown: text },
      ...options,
    };
    const response = await fetch(`${this.apiRoot}/bot${this.token}/sendRichMessage`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(payload),
      signal: AbortSignal.timeout(130_000),
    });
    const decoded = await response.json() as {
      ok: boolean;
      result?: Message;
      description?: string;
    };
    if (!response.ok || !decoded.ok || !decoded.result) {
      throw new Error(`Telegram sendRichMessage failed: ${decoded.description ?? response.status}`);
    }
    return sent(decoded.result);
  }

  async editReplyMarkup(chatId: number, messageId: number, markup: unknown): Promise<void> {
    await this.bot.api.editMessageReplyMarkup(chatId, messageId, { reply_markup: markup as never });
  }

  async answerCallbackQuery(id: string, options: Record<string, unknown> = {}): Promise<void> {
    await this.bot.api.answerCallbackQuery(id, options);
  }

  async deleteMessage(chatId: number, messageId: number): Promise<void> {
    await this.bot.api.deleteMessage(chatId, messageId);
  }

  async setReaction(chatId: number, messageId: number, emoji: string): Promise<void> {
    await this.bot.api.setMessageReaction(chatId, messageId, [
      { type: "emoji", emoji: emoji as never },
    ]);
  }

  async getFile(fileId: string): Promise<TelegramFile> {
    const file = await this.bot.api.getFile(fileId);
    return {
      fileId: file.file_id,
      fileUniqueId: file.file_unique_id,
      filePath: file.file_path,
      byteSize: file.file_size,
    };
  }

  async downloadFile(file: TelegramFile, signal?: AbortSignal): Promise<Uint8Array> {
    if (!file.filePath) throw new Error("Telegram file has no path");
    const response = await fetch(`${this.apiRoot}/file/bot${this.token}/${file.filePath}`, {
      signal,
    });
    if (!response.ok) throw new Error(`Telegram file download failed: HTTP ${response.status}`);
    return new Uint8Array(await response.arrayBuffer());
  }

  #input(value: UploadSource): string | InputFile {
    if (typeof value === "string") return value;
    return new InputFile(value instanceof Uint8Array ? value : value.path);
  }
}
