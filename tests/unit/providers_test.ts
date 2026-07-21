import { assertEquals, assertRejects } from "@std/assert";
import { DeepSeekProvider, GeminiProvider } from "../../src/ai/providers.ts";
import type { AiMessage, AiRequest } from "../../src/ai/types.ts";
import type { AppConfig } from "../../src/config.ts";

const user = (overrides: Partial<AiMessage> = {}): AiMessage => ({
  chatId: -1,
  messageId: 1,
  sentAt: 1_700_000_000,
  userId: 7,
  username: "测试用户",
  role: "user",
  messageType: "text",
  text: "你好",
  media: [],
  ...overrides,
});

const request = (messages: AiMessage[], overrides: Partial<AiRequest> = {}): AiRequest => ({
  sessionId: 9,
  provider: "gemini",
  model: "gemini-3-flash-preview",
  systemPrompt: "system",
  messages,
  newMessages: messages.slice(-1),
  historyRebuildLossy: false,
  allowCodeExecution: true,
  explicitCache: false,
  state: {},
  ...overrides,
});

const config = {
  geminiKey: "gemini-secret",
  deepSeekKey: "deepseek-secret",
  deepSeekBaseUrl: "https://deepseek.example",
} as AppConfig;

Deno.test("DeepSeek preserves raw assistant payload and rejects captionless video", async () => {
  let payload: Record<string, unknown> = {};
  const fetcher: typeof fetch = (_input, init) => {
    payload = JSON.parse(String(init?.body));
    return Promise.resolve(
      new Response(
        JSON.stringify({
          choices: [{ message: { role: "assistant", content: "回答", reasoning_content: "思考" } }],
          usage: {
            prompt_tokens: 12,
            prompt_cache_hit_tokens: 4,
            completion_tokens: 7,
            completion_tokens_details: { reasoning_tokens: 3 },
          },
        }),
        { status: 200 },
      ),
    );
  };
  const provider = new DeepSeekProvider(() => config, fetcher);
  const result = await provider.generate(request([user()], {
    provider: "deepseek",
    model: "deepseek-v4-flash",
  }));
  assertEquals(result.displayText, "回答");
  assertEquals(result.usage, {
    inputTokens: 12,
    cachedInputTokens: 4,
    outputTokens: 7,
    thinkingTokens: 3,
  });
  assertEquals((payload.thinking as Record<string, unknown>).type, "enabled");
  assertEquals(
    JSON.parse(new TextDecoder().decode(result.assistantPayload)).reasoning_content,
    "思考",
  );
  await assertRejects(
    () =>
      provider.generate(request([user({
        text: undefined,
        messageType: "video",
        media: [],
      })], { provider: "deepseek", model: "deepseek-v4-flash" })),
    Error,
    "当前模型不处理视频",
  );
});

Deno.test("Gemini Interactions compatibility errors fall back to generateContent", async () => {
  const calls: Array<{ url: string; headers: Headers; payload: Record<string, unknown> }> = [];
  const fetcher: typeof fetch = (input, init) => {
    calls.push({
      url: String(input),
      headers: new Headers(init?.headers),
      payload: JSON.parse(String(init?.body)),
    });
    if (calls.length === 1) {
      return Promise.resolve(
        new Response(JSON.stringify({ error: { message: "unsupported" } }), { status: 404 }),
      );
    }
    return Promise.resolve(
      new Response(
        JSON.stringify({
          candidates: [{ content: { role: "model", parts: [{ text: "fallback" }] } }],
          usageMetadata: { promptTokenCount: 10, candidatesTokenCount: 2 },
        }),
        { status: 200 },
      ),
    );
  };
  const provider = new GeminiProvider(() => config, fetcher);
  const result = await provider.generate(request([user()]));
  assertEquals(result.displayText, "fallback");
  assertEquals(calls.length, 2);
  assertEquals(calls[0]?.headers.get("Api-Revision"), "2026-05-20");
  assertEquals(calls[0]?.url.endsWith("/v1beta/interactions"), true);
  assertEquals(calls[1]?.url.includes(":generateContent"), true);
});

Deno.test("Gemini Interactions stores replayable model steps and remote state", async () => {
  const fetcher: typeof fetch = () =>
    Promise.resolve(
      new Response(
        JSON.stringify({
          id: "interaction-2",
          status: "completed",
          steps: [
            { type: "user_input", content: [{ type: "text", text: "hi" }] },
            { type: "model_output", content: [{ type: "text", text: "完成" }] },
          ],
          usage: { total_input_tokens: 20, total_cached_tokens: 5, total_output_tokens: 4 },
        }),
        { status: 200 },
      ),
    );
  const result = await new GeminiProvider(() => config, fetcher).generate(request([user()]));
  assertEquals(result.displayText, "完成");
  assertEquals(result.state.geminiInteractionId, "interaction-2");
  assertEquals(JSON.parse(new TextDecoder().decode(result.assistantPayload)), [
    { type: "model_output", content: [{ type: "text", text: "完成" }] },
  ]);
});
