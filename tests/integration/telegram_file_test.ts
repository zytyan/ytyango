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

Deno.test("local disk media uses JSON file URIs; remote uploads stream actual multipart bytes", async () => {
  const root = await Deno.makeTempDir();
  const bytes = new Uint8Array([0, 1, 2, 3, 255]);
  const path = `${root}/video #100% 中文.mp4`;
  const requests: {
    method: string;
    payload?: Record<string, unknown>;
    filename?: string;
    bytes?: Uint8Array;
  }[] = [];
  try {
    await Deno.writeFile(path, bytes);
    const fetcher: typeof fetch = async (input, options) => {
      const request = new Request(input, options);
      const method = new URL(request.url).pathname.split("/").at(-1)!;
      if (request.headers.get("content-type")?.startsWith("application/json")) {
        const payload = await request.json();
        requests.push({ method, payload });
      } else {
        const data = await request.formData();
        const file = [...data.values()].find((value) => value instanceof File) as File;
        requests.push({
          method,
          filename: file.name,
          bytes: new Uint8Array(await file.arrayBuffer()),
        });
      }
      return Response.json({
        ok: true,
        result: { message_id: 1, date: 1, chat: { id: 1, type: "private", first_name: "test" } },
      });
    };
    for (const host of ["127.0.0.1", "localhost", "[::1]"]) {
      const apiRoot = `http://${host}:8081`;
      const bot = new Bot("fake", { client: { apiRoot, fetch: fetcher as never } });
      const gateway = new GrammyTelegramGateway(bot, "fake", apiRoot);
      await gateway.sendVideo(1, { path });
      await gateway.sendAudio(1, { path });
      await gateway.sendPhoto(1, { path });
      for (const request of requests.slice(-3)) {
        const key = request.method.replace("send", "").toLowerCase();
        const uri = request.payload![key] as string;
        assertEquals(uri.startsWith("file:///"), true);
        assertEquals(new URL(uri).hash, "");
        assertEquals(await Deno.readFile(new URL(uri)), bytes);
      }
      await gateway.sendVideo(1, "existing-file-id");
      assertEquals(requests.at(-1)!.payload!.video, "existing-file-id");
    }
    const apiRoot = "https://api.telegram.org";
    const bot = new Bot("fake", { client: { apiRoot, fetch: fetcher as never } });
    const remote = new GrammyTelegramGateway(bot, "fake", apiRoot);
    await remote.sendVideo(1, { path });
    assertEquals(requests.at(-1)!.filename, "video #100% 中文.mp4");
    assertEquals(requests.at(-1)!.bytes, bytes);
    await remote.sendAudio(1, bytes);
    assertEquals(requests.at(-1)!.bytes, bytes);
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});
