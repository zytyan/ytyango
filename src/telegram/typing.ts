import type { TelegramPort } from "../shared/types.ts";

/** One in-flight action per request; cancellation stops renewal and aborts that action. */
export function startTyping(
  telegram: Pick<TelegramPort, "sendTyping">,
  chatId: number,
  topicId?: number,
  signal?: AbortSignal,
  intervalMs = 3000,
): () => void {
  const controller = new AbortController();
  const actionSignal = signal ? AbortSignal.any([signal, controller.signal]) : controller.signal;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const stop = () => {
    clearTimeout(timer);
    controller.abort();
  };
  const send = async () => {
    if (actionSignal.aborted) return;
    try {
      await telegram.sendTyping(chatId, topicId, actionSignal);
    } catch { /* Status delivery is best effort and cannot fail the AI request. */ }
    if (!actionSignal.aborted) timer = setTimeout(() => void send(), intervalMs);
  };
  actionSignal.addEventListener("abort", () => clearTimeout(timer), { once: true });
  void send();
  return stop;
}
