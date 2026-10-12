import { assertEquals, assertRejects, assertStringIncludes } from "@std/assert";
import type { Message } from "grammy/types";
import { albumIndex } from "../../src/ai/albums.ts";
import { temporaryDatabase } from "./helpers.ts";

Deno.test("album indexes only files, assembles in message order, isolates topics and rejects mixed or expired groups", async () => {
  const temporary = await temporaryDatabase();
  const now = Math.floor(Date.now() / 1000);
  const message = (id: number, extra: object = {}) =>
    ({
      message_id: id,
      date: now,
      chat: { id: -100, type: "supergroup" },
      caption: "ordinary private caption",
      media_group_id: "album",
      photo: [{
        file_id: `file-${id}`,
        file_unique_id: `uid-${id}`,
        width: 300,
        height: 300,
        file_size: 5,
      }],
      ...extra,
    }) as Message;
  const index = albumIndex(temporary.db, 10, 100);
  try {
    index.capture(message(2));
    index.capture(message(1));
    assertEquals(index.claim(message(1)), true);
    assertEquals(index.claim(message(2)), false);
    assertEquals(
      (await index.media(message(1), new AbortController().signal))!.map((item) => item.fileId),
      ["file-1", "file-2"],
    );
    assertEquals(
      (await index.media(message(2, { media_group_id: undefined }), new AbortController().signal))!
        .length,
      2,
    );
    const rows = temporary.db.all("SELECT * FROM album_files");
    assertEquals(JSON.stringify(rows).includes("caption"), false);
    assertEquals(temporary.db.all("SELECT * FROM ai_messages"), []);
    await assertRejects(() =>
      index.media(message(1, { message_thread_id: 5 }), new AbortController().signal)
    );
    index.capture(message(3, { photo: undefined, video: { file_id: "video" } }));
    const mixed = await assertRejects(
      () => index.media(message(1), new AbortController().signal),
      Error,
    );
    assertStringIncludes(mixed.message, "混合");
    await assertRejects(() => index.media(message(1), AbortSignal.abort()));
    index.cleanup(now + 7 * 86400);
    assertEquals(temporary.db.all("SELECT * FROM album_files"), []);
    await assertRejects(() => index.media(message(1), new AbortController().signal));
  } finally {
    await temporary.close();
  }
});
