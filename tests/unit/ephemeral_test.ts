import { assertEquals, assertRejects } from "@std/assert";
import { Bot, Context } from "grammy";
import type { Update, UserFromGetMe } from "grammy/types";
import {
  commandMenu,
  privateReplyOptions,
  recordCallbackReceived,
  registerCommandMenus,
  replyPrivately,
} from "../../src/telegram/ephemeral.ts";

Deno.test("ephemeral command declarations preserve public tools and private chat commands", () => {
  const groups = commandMenu(true), privateChats = commandMenu(false);
  assertEquals(groups.length, 21);
  assertEquals(new Set(groups.map((c) => c.command)).size, groups.length);
  for (const name of ["roll", "calc", "ocr", "downloadvideo", "downloadaudio", "getrank"]) {
    assertEquals(groups.find((c) => c.command === name)!.is_ephemeral, false);
  }
  assertEquals(groups.find((c) => c.command === "session_help")!.is_ephemeral, true);
  assertEquals(privateChats.every((c) => c.is_ephemeral === false), true);
});

Deno.test("menu registration replaces stale administrator and default command overrides", async () => {
  const bot = new Bot("fake");
  const calls: Record<string, unknown>[] = [];
  bot.api.config.use((_previous, method, payload) => {
    assertEquals(method, "setMyCommands");
    calls.push(payload as Record<string, unknown>);
    return Promise.resolve({ ok: true, result: true } as Awaited<ReturnType<typeof _previous>>);
  });
  await registerCommandMenus(bot.api);
  assertEquals(calls.length, 4);
  assertEquals(calls.map((c) => (c.scope as { type: string }).type), [
    "default",
    "all_private_chats",
    "all_group_chats",
    "all_chat_administrators",
  ]);
  for (const call of calls) {
    const group = ["all_group_chats", "all_chat_administrators"].includes(
      (call.scope as { type: string }).type,
    );
    assertEquals(call.commands, commandMenu(group));
  }
});

Deno.test("callback eligibility expires after 15 seconds; expiry cannot publish a private menu response", async () => {
  const bot = new Bot("fake");
  const user = { id: 7, is_bot: false, first_name: "user" } as const;
  const me = { id: 99, is_bot: true, first_name: "bot", username: "test_bot" } as const;
  let admin = false;
  bot.api.config.use((_previous, method) => {
    assertEquals(method, "getChatMember");
    return Promise.resolve(
      { ok: true, result: { user: me, status: admin ? "administrator" : "member" } } as Awaited<
        ReturnType<typeof _previous>
      >,
    );
  });
  const context = (privateMenu: boolean) =>
    new Context(
      {
        update_id: 1,
        callback_query: {
          id: "1234",
          from: user,
          chat_instance: "test",
          data: "x",
          message: {
            message_id: privateMenu ? 0 : 2,
            date: 1,
            chat: { id: -100, type: "supergroup", title: "test" },
            ...(privateMenu ? { ephemeral_message_id: 3, receiver_user: user } : {}),
          },
        },
      } as Update,
      bot.api,
      me as UserFromGetMe,
    );
  const normal = context(false), hidden = context(true);
  recordCallbackReceived(normal);
  recordCallbackReceived(hidden);
  assertEquals(await privateReplyOptions(normal), {
    ephemeral_message_parameters: { receiver_user_id: 7, callback_query_id: "1234" },
  });
  const expired = Date.now() + 16_000;
  assertEquals(await privateReplyOptions(normal, expired), { reply_parameters: { message_id: 2 } });
  assertEquals(await privateReplyOptions(hidden, expired), undefined);
  admin = true;
  Object.assign(hidden.callbackQuery!.message!, { is_topic_message: true, message_thread_id: 55 });
  assertEquals(await privateReplyOptions(hidden, expired), {
    message_thread_id: 55,
    ephemeral_message_parameters: { receiver_user_id: 7 },
  });
});

Deno.test("an ambiguous ephemeral send failure is never retried or sent publicly", async () => {
  const bot = new Bot("fake");
  const me = { id: 99, is_bot: true, first_name: "bot", username: "test_bot" } as const;
  let sends = 0;
  bot.api.config.use((_previous, method, payload) => {
    assertEquals(method, "sendMessage");
    assertEquals((payload as Record<string, unknown>).ephemeral_message_parameters, {
      receiver_user_id: 7,
    });
    sends++;
    return Promise.reject(new Error("response lost"));
  });
  const ctx = new Context(
    {
      update_id: 1,
      message: {
        message_id: 0,
        ephemeral_message_id: 3,
        date: Math.floor(Date.now() / 1000),
        from: { id: 7, is_bot: false, first_name: "user" },
        chat: { id: -100, type: "supergroup", title: "test" },
        text: "/session_help",
      },
    } as Update,
    bot.api,
    me as UserFromGetMe,
  );
  await assertRejects(() => replyPrivately(ctx, "private detail"), Error, "response lost");
  assertEquals(sends, 1);
});

Deno.test("unknown bot membership cannot turn a private query into a public response", async () => {
  const bot = new Bot("fake");
  const me = { id: 99, is_bot: true, first_name: "bot", username: "test_bot" } as const;
  let lookups = 0;
  bot.api.config.use((_previous, method) => {
    assertEquals(method, "getChatMember");
    lookups++;
    return Promise.reject(new Error("membership unavailable"));
  });
  const ctx = new Context(
    {
      update_id: 1,
      message: {
        message_id: 5,
        date: Math.floor(Date.now() / 1000),
        from: { id: 7, is_bot: false, first_name: "user" },
        chat: { id: -100, type: "supergroup", title: "test" },
        text: "/get_sysprompt",
      },
    } as Update,
    bot.api,
    me as UserFromGetMe,
  );
  assertEquals(await replyPrivately(ctx, "private prompt"), undefined);
  assertEquals(lookups, 1);
});
