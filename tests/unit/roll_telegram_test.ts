import { assertEquals, assertRejects } from "@std/assert";
import { Bot } from "grammy";
import type { Message, Update } from "grammy/types";
import { registerRoll } from "../../src/features/roll.ts";

const user = { id: 42, is_bot: false, first_name: "Test user" } as const;
const chat = { id: -10042, type: "supergroup", title: "Offline test" } as const;
const botUser = { id: 123456, is_bot: true, first_name: "Test bot", username: "test_bot" } as const;

async function harness(failDeletion = false) {
  const bot = new Bot("123456:test-token");
  registerRoll(bot);
  const calls: { method: string; payload: unknown }[] = [];
  bot.api.config.use((previous, method, payload) => {
    if (method === "getMe") {
      return Promise.resolve({
        ok: true,
        result: {
          ...botUser,
          can_join_groups: true,
          can_read_all_group_messages: true,
          supports_inline_queries: false,
        },
      } as Awaited<ReturnType<typeof previous>>);
    }
    // Compare the serialized wire payload, not grammY's helper class prototypes/undefined fields.
    calls.push({ method, payload: JSON.parse(JSON.stringify(payload)) });
    let result: unknown;
    switch (method) {
      case "getChatMember":
        result = { user: botUser, status: "member" };
        break;
      case "sendMessage":
        result = { message_id: 88, date: 1, chat, from: botUser, text: "Test reply" };
        break;
      case "answerCallbackQuery":
        result = true;
        break;
      case "deleteMessage":
        if (failDeletion) throw new Error("Deletion denied");
        result = true;
        break;
      default:
        throw new Error(`Unexpected API call: ${method}`);
    }
    // Typed mock boundary only. Never call previous: every API request is intercepted offline.
    return Promise.resolve({ ok: true, result } as Awaited<ReturnType<typeof previous>>);
  });
  await bot.init();
  return { bot, calls };
}

function command(text: string): Update {
  const name = text.split(" ")[0] ?? text;
  return {
    update_id: 1,
    message: {
      message_id: 10,
      date: 1,
      chat,
      from: user,
      text,
      entities: [{ type: "bot_command", offset: 0, length: name.length }],
    },
  };
}

function callback(withReply: boolean): Update {
  const message: Message = { message_id: 88, date: 1, chat, from: botUser, text: "Roll result" };
  if (withReply) {
    // Telegram omits nested reply_to_message; isolate the external recursive type at this fixture boundary.
    message.reply_to_message = {
      message_id: 10,
      date: 1,
      chat,
      from: user,
      text: "/roll",
    } as NonNullable<Message["reply_to_message"]>;
  }
  return {
    update_id: 2,
    callback_query: {
      id: "callback-1",
      from: user,
      chat_instance: "offline",
      data: "delme",
      message,
    },
  };
}

Deno.test("/roll Telegram command preserves reply identity and legacy button", async () => {
  const { bot, calls } = await harness();
  await bot.handleUpdate(command("/roll@test_bot 7 7"));
  assertEquals(calls, [{
    method: "sendMessage",
    payload: {
      chat_id: chat.id,
      text: "在7-7的roll点中，你掷出了7",
      reply_parameters: { message_id: 10 },
      reply_markup: { inline_keyboard: [[{ text: "删除该消息", callback_data: "delme" }]] },
    },
  }]);
});

Deno.test("/roll ignores commands for another bot", async () => {
  const { bot, calls } = await harness();
  await bot.handleUpdate(command("/roll@another_bot"));
  assertEquals(calls, []);
});

Deno.test("/roll Telegram invalid range returns a user-facing error", async () => {
  const { bot, calls } = await harness();
  await bot.handleUpdate(command("/roll 0"));
  assertEquals(calls, [{
    method: "getChatMember",
    payload: { chat_id: chat.id, user_id: botUser.id },
  }, {
    method: "sendMessage",
    payload: {
      chat_id: chat.id,
      text: "随机数范围无效或过大",
      reply_parameters: { message_id: 10 },
    },
  }]);
});

Deno.test("delme acknowledges then deletes response and original command", async () => {
  const { bot, calls } = await harness();
  await bot.handleUpdate(callback(true));
  assertEquals(calls, [
    { method: "answerCallbackQuery", payload: { callback_query_id: "callback-1" } },
    { method: "deleteMessage", payload: { chat_id: chat.id, message_id: 88 } },
    { method: "deleteMessage", payload: { chat_id: chat.id, message_id: 10 } },
  ]);
});

Deno.test("delme without original reply deletes only the response", async () => {
  const { bot, calls } = await harness();
  await bot.handleUpdate(callback(false));
  assertEquals(calls, [
    { method: "answerCallbackQuery", payload: { callback_query_id: "callback-1" } },
    { method: "deleteMessage", payload: { chat_id: chat.id, message_id: 88 } },
  ]);
});

Deno.test("delme does not delete the command when response deletion fails", async () => {
  const { bot, calls } = await harness(true);
  await assertRejects(() => bot.handleUpdate(callback(true)));
  assertEquals(calls, [
    { method: "answerCallbackQuery", payload: { callback_query_id: "callback-1" } },
    { method: "deleteMessage", payload: { chat_id: chat.id, message_id: 88 } },
  ]);
});

Deno.test("delme without a callback message only acknowledges", async () => {
  const { bot, calls } = await harness();
  await bot.handleUpdate({
    update_id: 3,
    callback_query: {
      id: "inline-callback",
      from: user,
      chat_instance: "offline",
      data: "delme",
      inline_message_id: "inline",
    },
  });
  assertEquals(calls, [{
    method: "answerCallbackQuery",
    payload: { callback_query_id: "inline-callback" },
  }]);
});

Deno.test("delme never deletes an ephemeral message through ordinary message ID zero", async () => {
  const { bot, calls } = await harness();
  const update = callback(true);
  update.callback_query!.message!.message_id = 0;
  Object.assign(update.callback_query!.message!, { ephemeral_message_id: 88, receiver_user: user });
  await bot.handleUpdate(update);
  assertEquals(calls, [{
    method: "answerCallbackQuery",
    payload: { callback_query_id: "callback-1" },
  }]);
});

Deno.test("CoC commands are not registered by the standalone roll feature", async () => {
  const { bot, calls } = await harness();
  for (const text of ["/cochelp", "/list_attr", "/del_attr", "/new_battle"]) {
    await bot.handleUpdate(command(text));
  }
  assertEquals(calls, []);
});
