import { assertEquals, assertStringIncludes } from "@std/assert";
import { createApplication } from "../../src/app.ts";
import { ConfigStore, parseConfig } from "../../src/config.ts";
import { temporaryDatabase } from "./helpers.ts";
import type { Message, Update } from "grammy/types";

Deno.test("traditional commands precede explicit AI, saved facts are user context, traditional replies alone do not trigger", async () => {
  const temporary = await temporaryDatabase();
  const originalFetch = globalThis.fetch;
  const chat = { id: -100, type: "supergroup", title: "test" } as const;
  const botUser = { id: 99, is_bot: true, first_name: "bot", username: "test_bot" } as const;
  const user = { id: 7, is_bot: false, first_name: "user" } as const;
  let nextId = 100;
  let modelCalls = 0;
  let fileDownloads = 0;
  const sent: Message[] = [];
  let lastBody: Record<string, unknown> = {};
  globalThis.fetch = (_input, options) => {
    const url = String(_input);
    if (url.includes("/file/botfake/")) {
      fileDownloads++;
      return Promise.resolve(new Response(new Uint8Array([1, 2, 3])));
    }
    if (url.endsWith("/sendRichMessage")) {
      const payload = JSON.parse(String(options?.body));
      const message = {
        message_id: nextId++,
        date: 1,
        chat,
        from: botUser,
        text: payload.rich_message.markdown,
      } as Message;
      sent.push(message);
      return Promise.resolve(Response.json({ ok: true, result: message }));
    }
    modelCalls++;
    lastBody = JSON.parse(String(options?.body));
    assertEquals(
      temporary.db.get<{ count: number }>("SELECT COUNT(*) AS count FROM traditional_facts")!
        .count > 0,
      true,
    );
    assertEquals(
      temporary.db.get<{ count: number }>("SELECT COUNT(*) AS count FROM ai_run_context")!.count >
        0,
      true,
    );
    return Promise.resolve(
      Response.json({
        candidates: [{
          content: { role: "model", parts: [{ text: "answer" }] },
          finishReason: "STOP",
        }],
        usageMetadata: { promptTokenCount: 3, candidatesTokenCount: 2 },
      }),
    );
  };
  try {
    const cfg = parseConfig(
      `database-path: ${temporary.path}\nbot-token: fake\nai-chats: [-100]\nsub2api-key: fake\n`,
    );
    temporary.db.run("INSERT INTO ai_chat_settings VALUES(-100,'sub2api','gemini-3.7-flash',0,1)");
    const app = await createApplication(
      temporary.db,
      new ConfigStore(cfg, "unused"),
      (bot) =>
        bot.api.config.use((_previous, method, payload) => {
          let result: unknown = true;
          if (method === "getMe") {
            result = {
              ...botUser,
              can_join_groups: true,
              can_read_all_group_messages: true,
              supports_inline_queries: true,
            };
          } else if (method === "getFile") {
            const id = (payload as { file_id: string }).file_id;
            result = { file_id: id, file_unique_id: id, file_path: `photos/${id}` };
          } else if (method === "sendMessage") {
            const message = {
              message_id: nextId++,
              date: 1,
              chat,
              from: botUser,
              text: (payload as { text: string }).text,
            } as Message;
            sent.push(message);
            result = message;
          } else if (!["setMessageReaction"].includes(method)) {
            throw new Error(`unexpected API ${method}`);
          }
          return Promise.resolve({ ok: true, result } as Awaited<ReturnType<typeof _previous>>);
        }),
    );
    const update = (id: number, text: string, reply?: Message, extra: object = {}): Update => ({
      update_id: id,
      message: {
        message_id: nextId++,
        date: 1,
        chat,
        from: user,
        text,
        reply_to_message: reply as never,
        // Ordinary supergroup replies can carry a thread ID without being forum topics.
        message_thread_id: reply?.message_id,
        entities: text.startsWith("/")
          ? [{ type: "bot_command", offset: 0, length: text.split(" ")[0]!.length }]
          : undefined,
        ...extra,
      },
    });
    await app.bot.handleUpdate(update(1, "ordinary chat"));
    assertEquals(temporary.db.all("SELECT * FROM ai_messages"), []);
    await app.bot.handleUpdate(update(2, "/roll 2 2"));
    assertEquals(modelCalls, 0);
    const traditional = sent.at(-1)!;
    await app.bot.handleUpdate(update(3, "hello", traditional));
    assertEquals(modelCalls, 0);
    await app.bot.handleUpdate(update(4, "@test_bot explain", traditional));
    assertEquals(modelCalls, 1);
    assertStringIncludes(JSON.stringify(lastBody.contents), "掷出了2");
    assertEquals(JSON.stringify(lastBody.systemInstruction).includes("掷出了2"), false);
    await app.bot.handleUpdate(update(5, "/roll 3 3 @test_bot"));
    assertEquals(modelCalls, 2);
    assertStringIncludes(JSON.stringify(lastBody.contents), "掷出了3");
    await app.bot.handleUpdate(update(6, "continue", sent.at(-1)!));
    assertEquals(modelCalls, 3);
    assertStringIncludes(JSON.stringify(lastBody.contents), "掷出了3");
    await app.bot.handleUpdate(update(7, "/calc 1/3+1/6 @test_bot"));
    assertEquals(modelCalls, 4);
    assertStringIncludes(JSON.stringify(lastBody.contents), "1/3+1/6 = 0.5");
    const albumMessage = (id: number, caption: string) =>
      ({
        message_id: id,
        date: Math.floor(Date.now() / 1000),
        chat,
        from: user,
        caption,
        media_group_id: "test-album",
        photo: [{ file_id: `photo-${id}`, file_unique_id: `uid-${id}`, width: 400, height: 400 }],
      }) as Message;
    const ordinary = albumMessage(901, "ordinary private album caption");
    await Promise.all([
      app.bot.handleUpdate({ update_id: 900, message: albumMessage(900, "@test_bot album") }),
      app.bot.handleUpdate({ update_id: 901, message: ordinary }),
      app.bot.handleUpdate({ update_id: 902, message: albumMessage(902, "") }),
    ]);
    assertEquals(modelCalls, 5);
    assertEquals(fileDownloads, 3);
    assertEquals(JSON.stringify(lastBody).includes("ordinary private album caption"), false);
    assertEquals(temporary.db.all("SELECT * FROM ai_message_media").length, 3);
    nextId = 1000;
    // Telegram may omit media_group_id in the embedded replied-to photo.
    await app.bot.handleUpdate(
      update(903, "@test_bot inspect all", { ...ordinary, media_group_id: undefined }),
    );
    assertEquals(modelCalls, 6);
    assertEquals(fileDownloads, 6);
    const ordinaryAnswer = sent.at(-1)!;
    const forum = { is_topic_message: true, message_thread_id: 55 };
    await app.bot.handleUpdate(update(904, "cross topic", ordinaryAnswer, forum));
    assertEquals(modelCalls, 6);
    await app.bot.handleUpdate(update(905, "@test_bot forum conversation", undefined, forum));
    assertEquals(modelCalls, 7);
    const forumAnswer = { ...sent.at(-1)!, ...forum };
    await app.bot.handleUpdate(update(906, "continue forum", forumAnswer, forum));
    assertEquals(modelCalls, 8);
    assertStringIncludes(JSON.stringify(lastBody.contents), "forum conversation");
    await app.bot.handleUpdate(
      update(907, "wrong topic", forumAnswer, { ...forum, message_thread_id: 56 }),
    );
    await app.bot.handleUpdate(update(908, "outside topic", forumAnswer));
    assertEquals(modelCalls, 8);
    await app.bot.handleUpdate(update(909, "@test_bot see documentation", undefined, {
      entities: [{ type: "text_link", offset: 14, length: 13, url: "https://example.com/manual" }],
    }));
    assertEquals(modelCalls, 9);
    assertStringIncludes(JSON.stringify(lastBody.contents), "https://example.com/manual");
    await app.bot.handleUpdate(update(910, "continue link discussion", sent.at(-1)!));
    assertEquals(modelCalls, 10);
    assertStringIncludes(JSON.stringify(lastBody.contents), "https://example.com/manual");
    app.flush();
  } finally {
    globalThis.fetch = originalFetch;
    await temporary.close();
  }
});
