import { assertEquals, assertStringIncludes } from "@std/assert";
import { createApplication } from "../../src/app.ts";
import { ConfigStore, parseConfig } from "../../src/config.ts";
import { StatsService } from "../../src/stats/service.ts";
import { temporaryDatabase } from "./helpers.ts";

Deno.test("automatic short links replace clean redirects, preserve failed originals and omit non-video download buttons", async () => {
  const temp = await temporaryDatabase();
  const original = globalThis.fetch;
  const chat = { id: -100, type: "supergroup", title: "test" } as const;
  const outputs: { text: string; reply_markup?: unknown }[] = [];
  const deleted: number[] = [];
  globalThis.fetch = (input) => {
    const url = String(input);
    if (!url.startsWith("https://b23.tv/")) throw new Error("unexpected paid or external request");
    return Promise.resolve(
      url.endsWith("expired")
        ? Response.json({ code: -404, message: "gone" })
        : new Response(null, {
          status: 302,
          headers: {
            location: url.endsWith("mall")
              ? "https://mall.bilibili.com/detail.html?itemsId=1"
              : "https://www.bilibili.com/video/BV1pmJy6mEre?p=1",
          },
        }),
    );
  };
  try {
    new StatsService(temp.db).setConfigField(-100, "auto_cvt_bili", true);
    const config = parseConfig(`database-path: ${temp.path}\nbot-token: fake\nai-chats: [-100]\n`);
    const app = await createApplication(temp.db, new ConfigStore(config, "unused"), (bot) => {
      bot.api.config.use((_previous, method, payload) => {
        if (method === "sendMessage") {
          outputs.push(payload as { text: string; reply_markup?: unknown });
        }
        if (method === "deleteMessage") {
          const id = (payload as { message_id: number }).message_id;
          deleted.push(id);
          if (id === 4) return Promise.reject(new Error("no delete permission"));
        }
        const result = method === "getMe"
          ? { id: 99, is_bot: true, first_name: "bot", username: "test_bot" }
          : method === "sendMessage"
          ? { chat, message_id: 10 + outputs.length, date: 1 }
          : true;
        return Promise.resolve({ ok: true, result } as Awaited<ReturnType<typeof _previous>>);
      });
    });
    for (const [index, code] of ["clean", "expired", "mall", "no-delete"].entries()) {
      await app.bot.handleUpdate({
        update_id: 980010 + index,
        message: {
          chat,
          message_id: index + 1,
          date: Math.floor(Date.now() / 1000),
          from: { id: 7, is_bot: false, first_name: "user" },
          text: "https://b23.tv/" + code,
        },
      });
    }
    assertEquals(outputs.length, 4);
    assertStringIncludes(outputs[0]!.text, "https://www.bilibili.com/video/BV1pmJy6mEre?p=1");
    assertStringIncludes(outputs[1]!.text, "上游返回 -404");
    assertEquals(outputs[2]!.reply_markup, undefined);
    assertEquals(deleted, [1, 3, 4]);
    assertEquals(temp.db.get<{ count: number }>("SELECT COUNT(*) AS count FROM ai_runs")!.count, 0);
    assertEquals(
      temp.db.get<{ count: number }>(
        "SELECT COUNT(*) AS count FROM traditional_facts WHERE feature='bilibili' AND status='success'",
      )!.count,
      3,
    );
    app.flush();
  } finally {
    globalThis.fetch = original;
    await temp.close();
  }
});
