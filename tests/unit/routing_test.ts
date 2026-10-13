import { assertEquals } from "@std/assert";
import { parseSessionRouting } from "../../src/ai/service.ts";
import { isAiTrigger } from "../../src/interaction.ts";

Deno.test("AI triggers follow Go command exclusions, UTF-8 limit and bot-reply rules", () => {
  assertEquals(isAiTrigger("/calc 1+1 @bot", "bot", true), false);
  assertEquals(isAiTrigger("@123", "bot", false), false);
  assertEquals(isAiTrigger("@newspaper", "bot", false), true);
  assertEquals(isAiTrigger("reply", "bot", false), false);
  assertEquals(isAiTrigger("reply", "bot", true), true);
  assertEquals(isAiTrigger("你好@bot", "bot", false), true);
  assertEquals(isAiTrigger("中".repeat(334), "bot", true), false);
  assertEquals(isAiTrigger("中".repeat(333), "bot", true), true);
});

Deno.test("AI explicit routing markers retain their documented precedence", () => {
  assertEquals(parseSessionRouting("continue @last @123 @new"), {
    createNew: true,
    ignoreCurrent: false,
  });
  assertEquals(parseSessionRouting("continue @last @123"), {
    createNew: false,
    ignoreCurrent: true,
  });
  assertEquals(parseSessionRouting("continue @123"), {
    createNew: false,
    ignoreCurrent: false,
    explicitSessionId: 123,
  });
});
