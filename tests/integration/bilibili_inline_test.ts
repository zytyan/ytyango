import { assertEquals, assertStringIncludes } from "@std/assert";
import { createApplication } from "../../src/app.ts";
import { ConfigStore, parseConfig } from "../../src/config.ts";
import { DownloadService } from "../../src/download/service.ts";
import { parseBilibiliInlineId } from "../../src/download/bilibili.ts";
import { FakeTelegram, FakeYoutubeRunner, temporaryDatabase } from "./helpers.ts";

Deno.test("inline Bilibili conversion works after imported 64-bit UIDs exceed JavaScript safe integers", async () => {
  const temp = await temporaryDatabase();
  const original = globalThis.fetch;
  const answers: {
    results: {
      id: string;
      title: string;
      input_message_content: { message_text: string };
      reply_markup?: { inline_keyboard: { text: string; callback_data: string }[][] };
    }[];
  }[] = [];
  const imported = 191315037158114041n;
  temp.db.run(
    "INSERT INTO bili_inline_results(uid,text,chat_id,msg_id,created_at) VALUES(?,?,?,?,?)",
    imported,
    "",
    0,
    0,
    1,
  );
  globalThis.fetch = () =>
    Promise.resolve(
      new Response(null, {
        status: 302,
        headers: { location: "https://www.bilibili.com/video/BV1pmJy6mEre?p=1&tracking=1" },
      }),
    );
  try {
    const cfg = parseConfig(`database-path: ${temp.path}\nbot-token: fake\n`);
    const app = await createApplication(temp.db, new ConfigStore(cfg, "unused"), (bot) => {
      bot.api.config.use((_previous, method, payload) => {
        if (method === "answerInlineQuery") answers.push(payload as typeof answers[number]);
        return Promise.resolve(
          {
            ok: true,
            result: method === "getMe"
              ? { id: 99, is_bot: true, first_name: "bot", username: "ytyan_bot" }
              : true,
          } as Awaited<ReturnType<typeof _previous>>,
        );
      });
    });
    await app.bot.handleUpdate({
      update_id: 980020,
      inline_query: {
        id: "inline-64bit",
        from: { id: 7, is_bot: false, first_name: "user" },
        query: "https://b23.tv/v62GPa7",
        offset: "",
        chat_type: "supergroup",
      },
    });
    assertEquals(answers.length, 1);
    assertEquals(answers[0]!.results.length, 2);
    const first = answers[0]!.results[0]!;
    assertStringIncludes(first.input_message_content.message_text, "BV1pmJy6mEre?p=1");
    assertEquals(first.id, `bv-${imported + 1n}`);
    assertEquals(
      first.reply_markup!.inline_keyboard[0]![0]!.callback_data,
      `il:bili:${(imported + 1n).toString(16)}`,
    );
    const uid = parseBilibiliInlineId(first.reply_markup!.inline_keyboard[0]![0]!.callback_data)!;
    const chat = { id: -100, type: "supergroup", title: "test" } as const;
    await app.bot.handleUpdate({
      update_id: 980021,
      message: {
        message_id: 21,
        date: Math.floor(Date.now() / 1000),
        chat,
        from: { id: 7, is_bot: false, first_name: "user" },
        via_bot: { id: 99, is_bot: true, first_name: "bot", username: "ytyan_bot" },
        text: first.input_message_content.message_text,
        reply_markup: first.reply_markup!,
      },
    });
    const service = new DownloadService(temp.db, new FakeTelegram(), new FakeYoutubeRunner());
    assertEquals(service.getInlineContext(uid, Math.floor(Date.now() / 1000)), {
      text: first.input_message_content.message_text,
      chatId: -100,
      messageId: 21,
    });
    assertEquals(
      service.getInlineContext(uid, Math.floor(Date.now() / 1000) + 31 * 86400),
      undefined,
    );
    app.flush();
  } finally {
    globalThis.fetch = original;
    await temp.close();
  }
});

Deno.test("inline Telegram answer failures propagate without a second raw fallback answer", async () => {
  const temp = await temporaryDatabase();
  let answers = 0;
  try {
    const config = parseConfig(`database-path: ${temp.path}\nbot-token: fake\n`);
    const app = await createApplication(temp.db, new ConfigStore(config, "unused"), (bot) => {
      bot.api.config.use((_previous, method) => {
        if (method === "answerInlineQuery") {
          answers++;
          return Promise.reject(new Error("Telegram unavailable"));
        }
        return Promise.resolve(
          {
            ok: true,
            result: { id: 99, is_bot: true, first_name: "bot", username: "ytyan_bot" },
          } as Awaited<ReturnType<typeof _previous>>,
        );
      });
    });
    const { assertRejects } = await import("@std/assert");
    await assertRejects(() =>
      app.bot.handleUpdate({
        update_id: 980022,
        inline_query: {
          id: "inline-failed",
          from: { id: 7, is_bot: false, first_name: "user" },
          query: "https://www.bilibili.com/video/BV1pmJy6mEre",
          offset: "",
          chat_type: "supergroup",
        },
      })
    );
    assertEquals(answers, 1);
    app.flush();
  } finally {
    await temp.close();
  }
});

for (
  const scenario of [
    "bound",
    "delivery-failure",
    "unbound",
    "expired",
    "invalid",
    "message",
  ] as const
) {
  Deno.test(`Bilibili download callback handles ${scenario} inline context without requiring callback.message`, async () => {
    const temp = await temporaryDatabase();
    const calls: { method: string; payload: Record<string, unknown> }[] = [];
    const uid = 191315037158114044n;
    const url = "https://www.bilibili.com/video/BV1pmJy6mEre?p=1";
    const timestamp = Math.floor(Date.now() / 1000);
    const chat = { id: -100, type: "supergroup", title: "test" } as const;
    try {
      temp.db.run(
        "INSERT INTO bili_inline_results(uid,text,chat_id,msg_id,created_at) VALUES(?,?,?,?,?)",
        uid,
        scenario === "unbound" ? "" : url,
        scenario === "unbound" ? 0 : -100,
        scenario === "unbound" ? 0 : 21,
        scenario === "expired" ? timestamp - 31 * 86400 : timestamp,
      );
      // Exercise routing and delivery with an existing Telegram file, without a process or network.
      temp.db.run(
        "INSERT INTO yt_dl_results(url,audio_only,resolution,file_id,title,description,uploader,upload_count) VALUES(?,0,720,?,'video','','uploader',0)",
        url,
        "cached-inline-video",
      );
      const config = parseConfig(`database-path: ${temp.path}\nbot-token: fake\n`);
      const app = await createApplication(temp.db, new ConfigStore(config, "unused"), (bot) => {
        bot.api.config.use((_previous, method, payload) => {
          calls.push({ method, payload: payload as Record<string, unknown> });
          if (method === "sendVideo" && scenario === "delivery-failure") {
            return Promise.reject(new Error("upload unavailable"));
          }
          return Promise.resolve(
            {
              ok: true,
              result: method === "getMe"
                ? { id: 99, is_bot: true, first_name: "bot", username: "ytyan_bot" }
                : method === "sendVideo" || method === "sendMessage"
                ? {
                  message_id: 22,
                  date: timestamp,
                  chat,
                  video: { file_id: "cached-inline-video" },
                }
                : true,
            } as Awaited<ReturnType<typeof _previous>>,
          );
        });
      });
      await app.bot.handleUpdate({
        update_id: 980030,
        callback_query: {
          id: "inline-download",
          from: { id: 7, is_bot: false, first_name: "user" },
          chat_instance: "test-chat-instance",
          data: scenario === "invalid" ? "il:bili:bad-id" : `il:bili:${uid.toString(16)}`,
          ...(scenario === "message"
            ? {
              message: {
                message_id: 41,
                date: timestamp,
                chat: { ...chat, id: -200 },
                text: url,
              },
            }
            : { inline_message_id: "opaque-inline-message-id" }),
        },
      });
      const answers = calls.filter((call) => call.method === "answerCallbackQuery");
      assertEquals(answers.length, 1);
      const videos = calls.filter((call) => call.method === "sendVideo");
      if (["bound", "delivery-failure", "message"].includes(scenario)) {
        assertEquals(answers[0]!.payload.text, "开始下载");
        assertEquals(videos.length, 1);
        assertEquals(videos[0]!.payload.video, "cached-inline-video");
        assertEquals(videos[0]!.payload.chat_id, scenario === "message" ? -200 : -100);
        assertEquals(videos[0]!.payload.reply_parameters, {
          message_id: scenario === "message" ? 41 : 21,
        });
        if (scenario === "delivery-failure") {
          const errors = calls.filter((call) => call.method === "sendMessage");
          assertEquals(errors.length, 1);
          assertEquals(errors[0]!.payload.chat_id, -100);
          assertEquals(errors[0]!.payload.reply_parameters, { message_id: 21 });
        }
      } else {
        assertEquals(videos.length, 0);
        assertEquals(calls.filter((call) => call.method === "sendMessage").length, 0);
        assertEquals(answers[0]!.payload.show_alert, true);
        assertStringIncludes(
          answers[0]!.payload.text as string,
          scenario === "unbound" ? "尚未收到" : "已过期",
        );
      }
      app.flush();
    } finally {
      await temp.close();
    }
  });
}
