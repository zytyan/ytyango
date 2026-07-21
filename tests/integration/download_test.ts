import { assertEquals } from "@std/assert";
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
