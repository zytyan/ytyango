import { assertEquals } from "@std/assert";
import { startTyping } from "../../src/telegram/typing.ts";

Deno.test("typing renews after transient status failure and stops without overlapping actions", async () => {
  let calls = 0;
  const renewed = Promise.withResolvers<void>();
  const telegram = {
    sendTyping(chatId: number, topicId?: number) {
      assertEquals([chatId, topicId], [-100, 55]);
      calls++;
      if (calls === 1) return Promise.reject(new Error("status unavailable"));
      renewed.resolve();
      return Promise.resolve();
    },
  };
  const stop = startTyping(telegram, -100, 55, undefined, 0);
  try {
    await renewed.promise;
  } finally {
    stop();
  }
  await new Promise((resolve) => setTimeout(resolve, 0));
  assertEquals(calls, 2);
});

Deno.test("typing aborts in-flight status on stop or request cancellation; pre-cancelled requests send nothing", async () => {
  for (const abortRequest of [false, true]) {
    const request = new AbortController();
    let calls = 0;
    let action: AbortSignal | undefined;
    const telegram = {
      sendTyping(_chatId: number, _topicId?: number, signal?: AbortSignal) {
        calls++;
        action = signal;
        return new Promise<void>((_resolve, reject) =>
          signal!.addEventListener("abort", () => reject(signal!.reason), { once: true })
        );
      },
    };
    const stop = startTyping(telegram, -100, undefined, request.signal, 0);
    assertEquals(calls, 1);
    if (abortRequest) request.abort();
    else stop();
    assertEquals(action?.aborted, true);
    await new Promise((resolve) => setTimeout(resolve, 0));
    assertEquals(calls, 1);
    stop();
    const stopCancelled = startTyping(telegram, -100, undefined, AbortSignal.abort(), 0);
    stopCancelled();
    assertEquals(calls, 1);
  }
});
