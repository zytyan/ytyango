import { assertEquals, assertStringIncludes } from "@std/assert";
import { createApplication } from "../../src/app.ts";
import { ConfigStore, parseConfig } from "../../src/config.ts";
import { StatsService } from "../../src/stats/service.ts";
import { temporaryDatabase } from "./helpers.ts";

Deno.test("automatic crypto exchange delivers declared UTC+8 date precision and persists the same explicit fact", async () => {
  const temp = await temporaryDatabase();
  const original = globalThis.fetch;
  let requests = 0;
  let output = "";
  const chat = { id: -100, type: "supergroup", title: "test" } as const;
  globalThis.fetch = (input) => {
    requests++;
    assertEquals(
      String(input),
      "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/usdt.min.json",
    );
    return Promise.resolve(Response.json({ date: "2026-10-12", usdt: { cny: 7 } }));
  };
  try {
    new StatsService(temp.db).setConfigField(-100, "auto_exchange", true);
    const config = parseConfig(`database-path: ${temp.path}\nbot-token: fake\n`);
    const app = await createApplication(temp.db, new ConfigStore(config, "unused"), (bot) => {
      bot.api.config.use((_previous, method, payload) => {
        const result = method === "getMe"
          ? { id: 99, is_bot: true, first_name: "bot", username: "test_bot" }
          : method === "sendMessage"
          ? { chat, message_id: 8, date: 1 }
          : true;
        if (method === "sendMessage") output = (payload as { text: string }).text;
        return Promise.resolve({ ok: true, result } as Awaited<ReturnType<typeof _previous>>);
      });
    });
    await app.bot.handleUpdate({
      update_id: 980002,
      message: {
        chat,
        message_id: 1,
        date: 1,
        from: { id: 7, is_bot: false, first_name: "user" },
        text: "2 USDT RMB",
      },
    });
    assertEquals(requests, 1);
    assertStringIncludes(output, "2.0000 USDT = 14.0000 CNY");
    assertStringIncludes(
      output,
      "2026-10-12 08:00:00 至 2026-10-13 08:00:00 前（UTC+8，来源仅精确到日）",
    );
    assertEquals(
      temp.db.get<{ context_text: string }>(
        "SELECT context_text FROM traditional_facts WHERE feature='exchange'",
      )!.context_text,
      output,
    );
    assertEquals(temp.db.get<{ count: number }>("SELECT COUNT(*) AS count FROM ai_runs")!.count, 0);
    app.flush();
  } finally {
    globalThis.fetch = original;
    await temp.close();
  }
});
