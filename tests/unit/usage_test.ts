import { assertEquals, assertStringIncludes } from "@std/assert";
import { formatUsage } from "../../src/ai/usage.ts";

const usage = {
  session_id: 9,
  model: "gemini-3-flash-preview",
  input_tokens: 161,
  output_tokens: 1109,
  cached_input_tokens: 0,
  input_message_count: 5,
  input_first_msg_id: 1,
  input_last_msg_id: 5,
};
Deno.test("usage separates exact reported tokens, reasoning and undocumented input reference", () => {
  const alert = formatUsage({
    ...usage,
    raw_payload: new TextEncoder().encode(JSON.stringify({
      usage: {
        total_input_tokens: 161,
        total_output_tokens: 485,
        total_thought_tokens: 624,
        raw_prompt_token: 1129,
      },
    })),
  });
  assertStringIncludes(alert, "输入（接口上报）: 161 Token");
  assertStringIncludes(alert, "输出: 1109（正文485+推理624） Token");
  assertStringIncludes(alert, "原始输入参考: 1129（口径未确认）");
  assertStringIncludes(alert, "协议: Interactions");
  assertEquals([...alert].length <= 200, true);
});

Deno.test("legacy usage stays readable without raw metadata and primary counts fit Telegram's alert limit", () => {
  const alert = formatUsage({
    ...usage,
    model: "model".repeat(40),
    raw_payload: new Uint8Array([0]),
    cached_input_tokens: 100,
  });
  assertStringIncludes(alert, "输出: 1109（含推理） Token");
  assertStringIncludes(alert, "缓存（输入内）: 100 Token");
  assertEquals([...alert].length <= 200, true);
});
