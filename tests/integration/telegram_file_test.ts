import { assertEquals, assertRejects } from "@std/assert";
import { Bot } from "grammy";
import { GrammyTelegramGateway } from "../../src/telegram/gateway.ts";

Deno.test("local Telegram files are confined to the bot directory and HTTP mode stays supported", async () => {
  const root = await Deno.makeTempDir();
  const original = globalThis.fetch;
  try {
    await Deno.mkdir(`${root}/fake/photos`, { recursive: true });
    const path = `${root}/fake/photos/image.jpg`;
    await Deno.writeFile(path, new Uint8Array([1, 2, 3]));
    const local = new GrammyTelegramGateway(new Bot("fake"), "fake", "http://127.0.0.1:8081", root);
    assertEquals(
      await local.downloadFile({ fileId: "image", filePath: path }),
      new Uint8Array([1, 2, 3]),
    );
    await Deno.writeFile(`${root}/other.jpg`, new Uint8Array([4]));
    await Deno.symlink(`${root}/other.jpg`, `${root}/fake/photos/escape.jpg`);
    await assertRejects(() =>
      local.downloadFile({ fileId: "image", filePath: `${root}/fake/photos/escape.jpg` })
    );
    await assertRejects(() =>
      local.downloadFile({ fileId: "image", filePath: path }, AbortSignal.abort())
    );
    const remote = new GrammyTelegramGateway(
      new Bot("fake"),
      "fake",
      "https://api.telegram.org",
      root,
    );
    await assertRejects(() => remote.downloadFile({ fileId: "image", filePath: path }));
    globalThis.fetch = () => Promise.resolve(new Response(new Uint8Array([7, 8])));
    assertEquals(
      await remote.downloadFile({ fileId: "image", filePath: "photos/image.jpg" }),
      new Uint8Array([7, 8]),
    );
  } finally {
    globalThis.fetch = original;
    await Deno.remove(root, { recursive: true });
  }
});
