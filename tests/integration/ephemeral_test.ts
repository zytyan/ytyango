import { assertEquals, assertStringIncludes } from "@std/assert";
import type { Message, Update } from "grammy/types";
import { createApplication } from "../../src/app.ts";
import { ConfigStore, parseConfig } from "../../src/config.ts";
import { AiRepository } from "../../src/ai/repository.ts";
import { StatsService } from "../../src/stats/service.ts";
import { temporaryDatabase } from "./helpers.ts";

async function fixture() {
  const temporary = await temporaryDatabase();
  const calls: { method: string; payload: Record<string, unknown> }[] = [];
  const chat = { id: -100, type: "supergroup", title: "test" } as const;
  const botUser = { id: 99, is_bot: true, first_name: "bot", username: "test_bot" } as const;
  const user = { id: 7, is_bot: false, first_name: "user" } as const;
  let admin = true;
  let nextId = 100;
  const cfg = parseConfig(
    `database-path: ${temporary.path}\nbot-token: fake\nai-chats: [-100]\ngod: 7\ngemini-key: fake\n`,
  );
  const app = await createApplication(temporary.db, new ConfigStore(cfg, "unused"), (bot) => {
    bot.api.config.use((_previous, method, raw) => {
      const payload = raw as Record<string, unknown>;
      calls.push({ method, payload });
      let result: unknown = true;
      if (method === "getMe") result = botUser;
      else if (method === "getChatMember") {
        result = { user: botUser, status: admin ? "administrator" : "member" };
      } else if (method === "sendMessage") {
        const privateParameters = payload.ephemeral_message_parameters;
        result = {
          chat,
          date: Math.floor(Date.now() / 1000),
          from: botUser,
          text: payload.text,
          ...(privateParameters
            ? { message_id: 0, ephemeral_message_id: nextId++, receiver_user: user }
            : { message_id: nextId++ }),
          reply_markup: payload.reply_markup,
        };
      } else if (
        ![
          "editEphemeralMessageText",
          "editEphemeralMessageReplyMarkup",
          "editMessageText",
          "editMessageReplyMarkup",
          "answerCallbackQuery",
        ].includes(method)
      ) {
        throw new Error(`Unexpected API ${method}`);
      }
      return Promise.resolve({ ok: true, result } as Awaited<ReturnType<typeof _previous>>);
    });
  });
  const update = (text: string, extra: object = {}): Update => ({
    update_id: nextId++,
    message: {
      chat,
      from: user,
      date: Math.floor(Date.now() / 1000),
      message_id: nextId++,
      text,
      entities: text.startsWith("/")
        ? [{ type: "bot_command", offset: 0, length: text.split(" ")[0]!.length }]
        : undefined,
      ...extra,
    },
  });
  const menu = (privateMenu = true, receiverId = 7): Message => ({
    chat,
    from: botUser,
    date: Math.floor(Date.now() / 1000),
    text: "menu",
    ...(privateMenu
      ? { message_id: 0, ephemeral_message_id: 51, receiver_user: { ...user, id: receiverId } }
      : { message_id: 51 }),
  });
  const callback = (data: string, message = menu()): Update => ({
    update_id: nextId++,
    callback_query: { id: "1234", from: user, chat_instance: "test", message, data },
  });
  return {
    ...temporary,
    app,
    calls,
    update,
    callback,
    menu,
    setAdmin: (value: boolean) => admin = value,
  };
}

Deno.test("management replies and input errors are private; successful tools remain public", async () => {
  const f = await fixture();
  try {
    for (
      const command of [
        "/session_id",
        "/session_help",
        "/ai_reaction off",
        "/get_sysprompt",
        "/sysprompt",
        "/sysprompt test prompt",
        "/reset_sysprompt",
        "/new_session",
        "/show_usage",
        "/model",
        "/change_model",
        "/chat_config",
        "/ocr",
        "/score",
        "/webp2png",
        "/calc bad(",
        "/roll 99999999999999999",
      ]
    ) {
      const before = f.calls.length;
      await f.app.bot.handleUpdate(
        f.update(command, { is_topic_message: true, message_thread_id: 55 }),
      );
      const sends = f.calls.slice(before).filter((c) => c.method === "sendMessage");
      assertEquals(sends.length, 1, command);
      assertEquals(
        sends[0]!.payload.ephemeral_message_parameters,
        { receiver_user_id: 7 },
        command,
      );
      assertEquals(sends[0]!.payload.message_thread_id, 55);
    }
    assertEquals(f.db.all("SELECT * FROM ai_runs"), []);
    assertEquals(f.db.all("SELECT * FROM traditional_facts WHERE reply_msg_id=0"), []);
    const confirmation = f.calls.find((c) => String(c.payload.text).includes("已设置当前话题"))!;
    assertStringIncludes(String(confirmation.payload.text), "影响");
    assertEquals(String(confirmation.payload.text).includes("test prompt"), false);
    for (const command of ["/calc 1+2", "/roll 2 2"]) {
      await f.app.bot.handleUpdate(f.update(command));
      assertEquals(f.calls.at(-1)!.payload.ephemeral_message_parameters, undefined);
    }
    f.setAdmin(false);
    await f.app.bot.handleUpdate(f.update("/session_help"));
    assertEquals(f.calls.at(-1)!.payload.ephemeral_message_parameters, undefined);
    await f.app.bot.handleUpdate(
      f.update("/session_help", { chat: { id: 7, type: "private", first_name: "user" } }),
    );
    assertEquals(f.calls.at(-1)!.payload.ephemeral_message_parameters, undefined);
  } finally {
    f.app.flush();
    await f.close();
  }
});

Deno.test("ephemeral command IDs never enter facts, sessions, albums or statistics; expiration grants no side effects", async () => {
  const f = await fixture();
  try {
    f.setAdmin(false);
    const hidden = { message_id: 0, ephemeral_message_id: 42 };
    await f.app.bot.handleUpdate(f.update("/ai_reaction off", hidden));
    assertEquals(f.calls.at(-1)!.payload.reply_parameters, { ephemeral_message_id: 42 });
    assertEquals(f.calls.at(-1)!.payload.ephemeral_message_parameters, { receiver_user_id: 7 });
    assertEquals(new AiRepository(f.db).reactionsEnabled(7), false);
    // The same ephemeral identifier may be reused. Both commands must execute independently.
    await f.app.bot.handleUpdate(f.update("/ai_reaction on", hidden));
    assertEquals(new AiRepository(f.db).reactionsEnabled(7), true);
    await f.app.bot.handleUpdate(f.update("@test_bot hidden AI", hidden));
    assertStringIncludes(String(f.calls.at(-1)!.payload.text), "仅支持");
    await f.app.bot.handleUpdate(f.update("/calc 1+2", hidden));
    assertStringIncludes(String(f.calls.at(-1)!.payload.text), "仅支持");
    await f.app.bot.handleUpdate(
      f.update("/ai_reaction off", { ...hidden, date: Math.floor(Date.now() / 1000) - 20 }),
    );
    assertEquals(new AiRepository(f.db).reactionsEnabled(7), true);
    f.app.flush();
    for (
      const table of [
        "ai_runs",
        "ai_messages",
        "traditional_facts",
        "album_index",
        "chat_stat_daily",
      ]
    ) {
      assertEquals(f.db.all(`SELECT * FROM ${table}`), [], table);
    }
    f.setAdmin(true);
    await f.app.bot.handleUpdate(
      f.update("/session_help", { ...hidden, date: Math.floor(Date.now() / 1000) - 20 }),
    );
    assertEquals(f.calls.at(-1)!.payload.reply_parameters, undefined);
    assertEquals(f.calls.at(-1)!.payload.ephemeral_message_parameters, { receiver_user_id: 7 });
  } finally {
    await f.close();
  }
});

Deno.test("private model and config menus use dedicated edits, verify receiver and permissions, and retain old public menus", async () => {
  const f = await fixture();
  try {
    const repository = new AiRepository(f.db);
    const original = repository.getSelection(-100);
    await f.app.bot.handleUpdate(f.callback("aimodel2:0:0", f.menu(true, 8)));
    assertEquals(repository.getSelection(-100), original);
    assertStringIncludes(String(f.calls.at(-1)!.payload.text), "不属于你");
    await f.app.bot.handleUpdate(f.callback("aimodel2:0:0"));
    const edit = f.calls.find((c) => c.method === "editEphemeralMessageText")!;
    assertEquals(edit.payload.ephemeral_message_id, 51);
    assertEquals(edit.payload.receiver_user_id, 7);
    assertEquals(edit.payload.message_id, undefined);
    assertStringIncludes(String(edit.payload.text), "影响全群");
    await f.app.bot.handleUpdate(f.callback("gcfg:auto_calculate:F"));
    assertEquals(new StatsService(f.db).config(-100).autoCalculate, true);
    const markup = f.calls.find((c) => c.method === "editEphemeralMessageReplyMarkup")!;
    assertEquals(markup.payload.ephemeral_message_id, 51);
    assertEquals(markup.payload.receiver_user_id, 7);
    await f.app.bot.handleUpdate(f.callback("gcfg:auto_calculate:T", f.menu(false)));
    assertEquals(new StatsService(f.db).config(-100).autoCalculate, false);
    assertEquals(f.calls.some((c) => c.method === "editMessageReplyMarkup"), true);
    await f.app.bot.handleUpdate(f.callback("aimodel2:0:0", f.menu(false)));
    assertEquals(f.calls.some((c) => c.method === "editMessageText"), true);
    // Receiver identity is not an administration grant.
    const update = f.callback("gcfg:auto_calculate:F");
    update.callback_query!.from.id = 8;
    update.callback_query!.message = f.menu(true, 8);
    f.setAdmin(false);
    await f.app.bot.handleUpdate(update);
    assertEquals(new StatsService(f.db).config(-100).autoCalculate, false);
    assertStringIncludes(String(f.calls.at(-1)!.payload.text), "管理员");
  } finally {
    await f.close();
  }
});
