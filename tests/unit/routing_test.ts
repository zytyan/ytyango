import { assertEquals } from "@std/assert";
import { parseSessionRouting } from "../../src/ai/service.ts";

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
