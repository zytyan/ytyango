import type { SentMessage, TelegramPort } from "../shared/types.ts";

/** Best-effort status delivery never retries or changes the download's outcome. */
export async function withDownloadNotice<T>(
  telegram: Pick<TelegramPort, "sendMessage" | "deleteMessage">,
  chatId: number,
  requestId: number,
  operation: () => Promise<T>,
  signal?: AbortSignal,
  delayMs = 1000,
): Promise<T> {
  let notice: Promise<SentMessage | undefined> | undefined;
  const timer = setTimeout(() => {
    if (signal?.aborted) return;
    notice = telegram.sendMessage(chatId, "正在下载，请稍等", {
      reply_parameters: { message_id: requestId },
    }).catch(() => undefined);
  }, delayMs);
  try {
    signal?.throwIfAborted();
    return await operation();
  } finally {
    clearTimeout(timer);
    // An in-flight send can finish after the download; delete its eventual message too.
    const message = await notice;
    if (message) {
      try {
        await telegram.deleteMessage(message.chatId, message.messageId);
      } catch { /* Do not retry Telegram side effects or hide a successful download. */ }
    }
  }
}
