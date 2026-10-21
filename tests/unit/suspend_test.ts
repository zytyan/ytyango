import { assertEquals } from "@std/assert";
import { parseSuspension } from "../../src/ai/suspend.ts";

Deno.test("suspend accepts positive duration units, permanent and off; rejects malformed and overflowing input", () => {
  for (
    const [input, expected] of [
      ["", null],
      ["off", "off"],
      ["10s", 10],
      ["1m", 60],
      ["2h", 7200],
      ["1d", 86400],
    ] as const
  ) assertEquals(parseSuspension(input), expected);
  for (const input of ["0s", "-1m", "1.5h", "10", "1m 2h", "OFF", "1w", "999999999999999d"]) {
    assertEquals(parseSuspension(input), undefined, input);
  }
});
