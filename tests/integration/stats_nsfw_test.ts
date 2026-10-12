import { assertEquals, assertRejects } from "@std/assert";
import type { AppConfig } from "../../src/config.ts";
import { NsfwService } from "../../src/nsfw/service.ts";
import { StatsService } from "../../src/stats/service.ts";
import { FakeTelegram, temporaryDatabase } from "./helpers.ts";

const config = {
  contentModerator: { accountId: "account", apiToken: "key" },
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

Deno.test("Cloudflare collects safe anime, ignores safe photos, classifies every album item and rejects invalid choices", async () => {
  const temporary = await temporaryDatabase();
  const telegram = new FakeTelegram();
  const choices = [["0", "yes"], ["0", "no"], ["6", "no"], ["4", "yes"], ["oops", "no"]];
  let calls = 0;
  const fetcher: typeof fetch = (url, options) => {
    assertEquals(
      String(url),
      "https://api.cloudflare.com/client/v4/accounts/account/ai/run/@cf/cloudflare/clef-flash",
    );
    assertEquals(new Headers(options?.headers).get("authorization"), "Bearer key");
    const body = JSON.parse(String(options?.body));
    assertEquals(body.images[0].content_type, "image/jpeg");
    assertEquals(Object.keys(body.questions.microsoft_severity.criteria), ["0", "2", "4", "6"]);
    const [severity, anime] = choices[calls++]!;
    return Promise.resolve(
      Response.json({
        success: true,
        result: {
          answers: {
            microsoft_severity: { type: "choice", choice: severity },
            anime_illustration: { type: "choice", choice: anime },
          },
        },
      }),
    );
  };
  try {
    const service = new NsfwService(
      temporary.db,
      telegram,
      () => config,
      () => {},
      fetcher,
      (bytes) => Promise.resolve(bytes),
    );
    for (let index = 0; index < 4; index++) {
      await service.detectAndReply({
        chatId: -100,
        messageId: index,
        mediaGroupId: "album",
        hasMediaSpoiler: false,
        photo: { fileId: `file${index}`, fileUniqueId: `uid${index}`, width: 640, height: 480 },
      });
    }
    assertEquals(calls, 4);
    assertEquals(Boolean(service.getPicture("uid0")), true);
    assertEquals(Boolean(service.getPicture("uid1")), false);
    assertEquals(Boolean(service.getPicture("uid2")), true);
    assertEquals(Boolean(service.getPicture("uid3")), true);
    assertEquals(telegram.calls.filter((call) => call.method === "message").length, 1);
    await assertRejects(
      () => service.score({ fileId: "bad", fileUniqueId: "bad", width: 640, height: 480 }),
      Error,
      "invalid",
    );
    assertEquals(service.getPicture("bad"), undefined);
  } finally {
    await temporary.close();
  }
});

Deno.test("Cloudflare severity, gallery and rating triggers stay compatible", async () => {
  const temporary = await temporaryDatabase();
  const telegram = new FakeTelegram();
  const fetcher: typeof fetch = () =>
    Promise.resolve(
      new Response(
        JSON.stringify({
          success: true,
          result: {
            answers: {
              microsoft_severity: { type: "choice", choice: "6" },
              anime_illustration: { type: "choice", choice: "no" },
            },
          },
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );
  try {
    const nsfw = new NsfwService(
      temporary.db,
      telegram,
      () => config,
      () => {},
      fetcher,
      (bytes) => Promise.resolve(bytes),
    );
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
