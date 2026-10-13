import { assertEquals, assertRejects } from "@std/assert";
import { withDownloadNotice } from "../../src/download/notice.ts";
import type { SentMessage } from "../../src/shared/types.ts";

Deno.test("fast downloads skip the wait notice", async () => {
  let calls = 0;
  const telegram = {
    sendMessage: () => {
      calls++;
      return Promise.reject(new Error("unexpected send"));
    },
    deleteMessage: () => {
      calls++;
      return Promise.resolve();
    },
  };
  assertEquals(await withDownloadNotice(telegram, -100, 7, () => Promise.resolve(42)), 42);
  assertEquals(calls, 0);
});

Deno.test("private download notices use the ephemeral ID for cleanup without deleting message zero", async () => {
  const started = Promise.withResolvers<void>(), download = Promise.withResolvers<number>();
  let removed = 0;
  const telegram = {
    sendMessage: () => Promise.reject(new Error("public notice forbidden")),
    deleteMessage: () => Promise.reject(new Error("ordinary deletion forbidden")),
  };
  const message = { chatId: -100, messageId: 0, raw: { ephemeral_message_id: 9 } };
  const result = withDownloadNotice(telegram, -100, 7, () => download.promise, undefined, 0, {
    send() {
      started.resolve();
      return Promise.resolve(message);
    },
    remove(sent) {
      assertEquals(sent, message);
      removed++;
      return Promise.resolve();
    },
  });
  await started.promise;
  download.resolve(42);
  assertEquals(await result, 42);
  assertEquals(removed, 1);
});

Deno.test("slow download notices retain reply identity and clean up even after failure or a late send", async () => {
  for (const fails of [false, true]) {
    const started = Promise.withResolvers<void>();
    const download = Promise.withResolvers<number>();
    const sent = Promise.withResolvers<SentMessage>();
    let sends = 0, deletes = 0, operations = 0;
    const telegram = {
      sendMessage(chatId: number, text: string, options?: Record<string, unknown>) {
        sends++;
        assertEquals(chatId, -100);
        assertEquals(text, "正在下载，请稍等");
        assertEquals(options, { reply_parameters: { message_id: 7 } });
        started.resolve();
        return sent.promise;
      },
      deleteMessage(chatId: number, messageId: number) {
        deletes++;
        assertEquals([chatId, messageId], [-100, 99]);
        return Promise.resolve();
      },
    };
    const result = withDownloadNotice(
      telegram,
      -100,
      7,
      () => {
        operations++;
        return download.promise;
      },
      undefined,
      0,
    );
    const outcome = fails ? assertRejects(() => result, Error, "download failed") : result;
    await started.promise;
    if (fails) download.reject(new Error("download failed"));
    else download.resolve(42);
    sent.resolve({ chatId: -100, messageId: 99, raw: {} });
    if (fails) await outcome;
    else assertEquals(await outcome, 42);
    assertEquals([sends, deletes, operations], [1, 1, 1]);
  }
});

Deno.test("notice errors never retry or fail the download; pre-cancelled requests do no work", async () => {
  const started = Promise.withResolvers<void>(), download = Promise.withResolvers<number>();
  let sends = 0, operations = 0;
  const telegram = {
    sendMessage() {
      sends++;
      started.resolve();
      return Promise.reject(new Error("notice unavailable"));
    },
    deleteMessage: () => Promise.resolve(),
  };
  const operation = () => {
    operations++;
    return download.promise;
  };
  const result = withDownloadNotice(telegram, -100, 7, operation, undefined, 0);
  await started.promise;
  download.resolve(42);
  assertEquals(await result, 42);
  await assertRejects(() => withDownloadNotice(telegram, -100, 7, operation, AbortSignal.abort()));
  assertEquals([sends, operations], [1, 1]);
});
