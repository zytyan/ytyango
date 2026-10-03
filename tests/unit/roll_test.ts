import { assertEquals, assertThrows } from "@std/assert";
import { parseRollRange, roll } from "../../src/features/roll.ts";

const examples: Array<{ args: string[]; expected: [number, number] }> = [
  { args: [], expected: [1, 20] },
  { args: ["6"], expected: [1, 6] },
  { args: ["+6"], expected: [1, 6] },
  { args: ["5", "2"], expected: [2, 5] },
  { args: ["-5", "-2"], expected: [-5, -2] },
  { args: ["7", "7"], expected: [7, 7] },
  { args: ["abc"], expected: [1, 20] },
  { args: ["2abc"], expected: [1, 20] },
  { args: ["bad", "4"], expected: [1, 4] },
  { args: ["2", "5", "ignored"], expected: [2, 5] },
  {
    args: ["9007199254740990", "9007199254740991"],
    expected: [9007199254740990, 9007199254740991],
  },
];

for (const { args, expected } of examples) {
  Deno.test(`/roll parses ${JSON.stringify(args)}`, () => {
    assertEquals(parseRollRange(args), expected);
  });
}

for (const args of [["0"], ["-1"], ["9007199254740992"], ["0", "281474976710655"]]) {
  Deno.test(`/roll rejects unsafe range ${JSON.stringify(args)}`, () => {
    assertThrows(() => parseRollRange(args));
  });
}

Deno.test("/roll includes both bounds and preserves reply wording", () => {
  assertEquals(roll(["2", "5"], () => 0), "在2-5的roll点中，你掷出了2");
  assertEquals(roll(["2", "5"], (maximum) => maximum - 1), "在2-5的roll点中，你掷出了5");
  assertEquals(roll(["7", "7"], () => 0), "在7-7的roll点中，你掷出了7");
});

Deno.test("/roll validates random sample", () => {
  assertThrows(() => roll([], () => 20));
  assertThrows(() => roll([], () => -1));
  assertThrows(() => roll([], () => 0.5));
});

Deno.test("/roll production randomness stays within bounds", () => {
  for (let index = 0; index < 50; index++) {
    assertEquals(roll(["7", "7"]), "在7-7的roll点中，你掷出了7");
  }
});
