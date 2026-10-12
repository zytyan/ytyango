import { assertEquals, assertRejects } from "@std/assert";
import { AiRepository } from "../../src/ai/repository.ts";
import { AiService, type IncomingAiMessage } from "../../src/ai/service.ts";
import { createAiGenerator } from "../../src/ai/providers.ts";
import { ContentAddressedMediaStore } from "../../src/ai/media_store.ts";
import { parseConfig } from "../../src/config.ts";
import { FakeTelegram, temporaryDatabase } from "./helpers.ts";

Deno.test("media preparation failure never calls a provider or creates a partial request", async () => {
  const temporary = await temporaryDatabase();
  try {
    let calls = 0;
    const telegram = new FakeTelegram();
    let files = 0;
    telegram.downloadFile = () =>
      ++files === 1
        ? Promise.resolve(new Uint8Array([1, 2, 3]))
        : Promise.reject(new Error("file unavailable"));
    const service = new AiService(
      new AiRepository(temporary.db),
      () => {
        calls++;
        throw new Error("unexpected model call");
      },
      telegram,
      new ContentAddressedMediaStore(`${temporary.path}.media`),
      () => parseConfig(`database-path: ${temporary.path}\nai-chats: [-100]\n`),
      { id: 99, firstName: "bot", username: "bot" },
    );
    await assertRejects(() =>
      service.handle({
        chatId: -100,
        chatName: "test",
        chatType: "supergroup",
        messageId: 12,
        sentAt: 1,
        userId: 7,
        username: "user",
        text: "@bot",
        completeAlbum: true,
        media: [{ fileId: "photo", kind: "image", mimeType: "image/jpeg" }, {
          fileId: "missing",
          kind: "image",
          mimeType: "image/jpeg",
        }],
      })
    );
    assertEquals(calls, 0);
    assertEquals(temporary.db.all("SELECT * FROM ai_runs"), []);
    assertEquals(telegram.calls.some((call) => call.method === "message"), true);
  } finally {
    await temporary.close();
  }
});

Deno.test("SDK result survives Telegram failure; redelivery neither calls model nor counts tokens twice", async () => {
  const temporary = await temporaryDatabase();
  try {
    const repository = new AiRepository(temporary.db);
    repository.setChatModel(-100, "sub2api", "gemini-3.7-flash", 1);
    const config = parseConfig(
      `database-path: ${temporary.path}\nai-chats: [-100]\nsub2api-key: mock-key\n`,
    );
    let providerCalls = 0;
    const generate = createAiGenerator(() => config, () => {
      const pending = temporary.db.get<{ status: string }>(
        "SELECT status FROM ai_runs ORDER BY id DESC LIMIT 1",
      );
      assertEquals(pending?.status, "pending");
      providerCalls++;
      return Promise.resolve(
        new Response(
          JSON.stringify({
            candidates: [{
              content: { role: "model", parts: [{ text: "done", thoughtSignature: "AQID" }] },
              finishReason: "STOP",
            }],
            usageMetadata: {
              promptTokenCount: 11,
              candidatesTokenCount: 3,
              thoughtsTokenCount: 2,
              cachedContentTokenCount: 4,
            },
          }),
          { headers: { "content-type": "application/json" } },
        ),
      );
    });
    const telegram = new FakeTelegram();
    const sendRich = telegram.sendRichMessage.bind(telegram);
    telegram.sendRichMessage = () => Promise.reject(new Error("Telegram unavailable"));
    const service = new AiService(
      repository,
      generate,
      telegram,
      new ContentAddressedMediaStore(`${temporary.path}.media`),
      () => config,
      { id: 99, firstName: "bot", username: "bot" },
    );
    const incoming: IncomingAiMessage = {
      chatId: -100,
      chatName: "group",
      chatType: "supergroup",
      messageId: 10,
      sentAt: 2,
      userId: 7,
      username: "user",
      text: "question",
      media: [],
    };
    await assertRejects(() => service.handle(incoming), Error, "Telegram unavailable");
    assertEquals(providerCalls, 1);
    const run = temporary.db.get<{ id: number }>("SELECT id FROM ai_runs");
    assertEquals(repository.getRun(run!.id)?.status, "delivery_failed");
    assertEquals(repository.loadProviderState(1), {});
    telegram.sendRichMessage = sendRich;
    await service.handle(incoming);
    await service.handle(incoming);
    assertEquals(providerCalls, 1);
    assertEquals(repository.getRun(run!.id)?.status, "delivered");
    assertEquals(repository.getRun(run!.id)?.assistant_payload_format, "vercel-ai-messages-v1");
    assertEquals(
      temporary.db.get(
        "SELECT provider,model,total_input_tokens,total_output_tokens,total_cached_input_tokens FROM ai_sessions",
      ),
      {
        provider: "sub2api",
        model: "gemini-3.7-flash",
        total_input_tokens: 11,
        total_output_tokens: 5,
        total_cached_input_tokens: 4,
      },
    );
    assertEquals(telegram.calls.filter((call) => call.method === "rich").length, 1);
    assertEquals(telegram.calls.find((call) => call.method === "rich")?.options?.reply_parameters, {
      message_id: 10,
      allow_sending_without_reply: true,
    });
    assertEquals(repository.findSessionByMessage(-100, 100)?.provider, "sub2api");
  } finally {
    await temporary.close();
  }
});
