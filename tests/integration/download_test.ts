import { assertEquals, assertRejects } from "@std/assert";
import { DownloadService } from "../../src/download/service.ts";
import { FakeTelegram, FakeYoutubeRunner, temporaryDatabase } from "./helpers.ts";

Deno.test("same download runs and uploads once while every requester receives media", async () => {
  const temporary = await temporaryDatabase();
  const telegram = new FakeTelegram();
  const runner = new FakeYoutubeRunner();
  try {
    const service = new DownloadService(temporary.db, telegram, runner);
    const key = { url: "https://youtube.com/watch?v=test", audioOnly: false, resolution: 720 };
    await Promise.all([
      service.deliver({ key, chatId: -100, messageId: 1, userId: 1, userName: "A" }),
      service.deliver({ key, chatId: -100, messageId: 2, userId: 2, userName: "B" }),
    ]);
    assertEquals(runner.downloads, 1);
    const videos = telegram.calls.filter((call) => call.method === "video");
    assertEquals(videos.length, 2);
    assertEquals(typeof videos[0]?.media, "object");
    assertEquals(videos[1]?.media, "uploaded-video");
    assertEquals(runner.arguments.includes("res:720"), false);
    assertEquals(runner.arguments.some((value) => value.includes("res:720")), true);
    assertEquals(
      temporary.db.get<{ upload_count: number }>("SELECT upload_count FROM yt_dl_results")
        ?.upload_count,
      1,
    );
  } finally {
    await temporary.close();
  }
});

Deno.test("Bilibili audio uses account JSON and survives optional metadata failure", async () => {
  const temporary = await temporaryDatabase();
  const telegram = new FakeTelegram();
  let workDir = "";
  const runner = {
    async run(command: string, args: readonly string[], options: { cwd?: string }) {
      assertEquals(command, "/usr/bin/bbdown-go");
      assertEquals(args.includes("--audio-only"), true);
      assertEquals(args[1], "/account.json");
      workDir = options.cwd!;
      await Deno.writeFile(`${workDir}/audio.m4a`, new Uint8Array([1]));
      return {
        code: 0,
        stdout: new TextEncoder().encode(JSON.stringify({ audio_path: "audio.m4a" })),
        stderr: new Uint8Array(),
      };
    },
  };
  const fetcher: typeof fetch = () => Promise.reject(new Error("metadata unavailable"));
  try {
    const service = new DownloadService(
      temporary.db,
      telegram,
      runner,
      fetcher,
      () => Promise.resolve("/account.json"),
    );
    await service.deliver({
      key: { url: "https://bilibili.com/video/BV1", audioOnly: true, resolution: 0 },
      chatId: -100,
      messageId: 1,
      userId: 1,
      userName: "A",
    });
    assertEquals(telegram.calls.filter((call) => call.method === "audio").length, 1);
    await assertRejects(() => Deno.stat(workDir), Deno.errors.NotFound);
  } finally {
    await temporary.close();
  }
});
