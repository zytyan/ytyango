import { assertEquals } from "@std/assert";
import type { AppConfig } from "../../src/config.ts";
import { NsfwService } from "../../src/nsfw/service.ts";
import { StatsService } from "../../src/stats/service.ts";
import { FakeTelegram, temporaryDatabase } from "./helpers.ts";

const config = {
  contentModerator: { endpoint: "https://azure.example", apiKey: "key" },
} as AppConfig;

Deno.test("statistics count graphemes and timezone buckets transactionally", async () => {
  const temporary = await temporaryDatabase();
  try {
    const stats = new StatsService(temporary.db);
    temporary.db.run("INSERT INTO chat_cfg VALUES(?,0,0,0,0,0,0,?)", -100, -5 * 3600);
    stats.record({
      chatId: -100,
      messageId: 55,
      sentAt: 1_700_000_000,
      user: { id: 7, firstName: "测试" },
      text: "A👨‍👩‍👧‍👦",
      photo: true,
      video: false,
      sticker: false,
      forwarded: true,
    });
    stats.flush();
    const user = temporary.db.get<{ message_count: number; message_length: number }>(
      "SELECT message_count,message_length FROM chat_stat_user_daily WHERE chat_id=-100 AND user_id=7",
    );
    assertEquals(user, { message_count: 1, message_length: 2 });
    const bucket = temporary.db.get<{ message_count: number; first_msg_id: number }>(
      "SELECT message_count,first_msg_id FROM chat_stat_bucket_daily WHERE chat_id=-100 AND message_count=1",
    );
    assertEquals(bucket, { message_count: 1, first_msg_id: 55 });
  } finally {
    await temporary.close();
  }
});

Deno.test("NSFW severity, gallery and V9 rating triggers stay compatible", async () => {
  const temporary = await temporaryDatabase();
  const telegram = new FakeTelegram();
  const fetcher: typeof fetch = () =>
    Promise.resolve(
      new Response(
        JSON.stringify({
          categoriesAnalysis: [{ category: "Sexual", severity: 6 }],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );
  try {
    const nsfw = new NsfwService(temporary.db, telegram, () => config, () => {}, fetcher);
    const severity = await nsfw.detectAndReply({
      chatId: -100,
      messageId: 9,
      hasMediaSpoiler: false,
      photo: { fileId: "file", fileUniqueId: "unique", width: 640, height: 480 },
    });
    assertEquals(severity, 6);
    assertEquals(telegram.calls.filter((call) => call.method === "message").length, 1);
    assertEquals(nsfw.rate("unique", 1, 2).changed, true);
    assertEquals(nsfw.rate("unique", 2, 6).changed, true);
    assertEquals(nsfw.getPicture("unique"), {
      fileId: "file",
      botRate: 6,
      userRate: 4,
      sum: 8,
      count: 2,
    });
  } finally {
    await temporary.close();
  }
});
