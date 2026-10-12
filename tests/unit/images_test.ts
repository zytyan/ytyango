import { assertEquals, assertThrows } from "@std/assert";
import { fishParts, isFish } from "../../src/features/images.ts";

Deno.test("fish aliases preserve segment order and bound image width", () => {
  assertEquals(fishParts("サカ班basu魚"), [0, 1, 2, 3]);
  assertEquals(isFish("萨卡班甲鱼"), true);
  assertEquals(isFish("普通消息"), false);
  assertThrows(() => fishParts("鱼".repeat(101)), Error, "100");
});
