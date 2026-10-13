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
Deno.test("usage restores the Go layout without changing totals or showing raw diagnostics", () => {
  const alert = formatUsage({
    ...usage,
    raw_payload: JSON.stringify({ usage: { raw_prompt_token: 1129 } }),
  });
  assertEquals(
    alert,
    "ID: 9\n消息: 5（1-5）\n模型: gemini-3-flash-preview\n输入: 0.16 ktoken\n输出: 1.11 ktoken\n缓存: 0.00 ktoken\n缓存到期：无显式缓存",
  );
  assertEquals(usage.input_tokens, 161);
});

Deno.test("usage retains Shanghai cache expiry, missing history and the Unicode alert limit", () => {
  const alert = formatUsage({ ...usage, input_message_count: 0, cache_expire_at: 1 });
  assertStringIncludes(alert, "消息: 未记录");
  assertStringIncludes(alert, "缓存到期：1970-01-01 08:00:01");
  const long = formatUsage({ ...usage, model: "😀".repeat(250) });
  assertEquals([...long].length, 200);
  assertEquals(long.endsWith("…"), true);
});
