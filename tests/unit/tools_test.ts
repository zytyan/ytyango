import { assertEquals, assertRejects, assertThrows } from "@std/assert";
import { calculate, formatCalculation, isCalculation } from "../../src/features/calculator.ts";
import { exchange, ocr, parseExchange } from "../../src/features/tools.ts";
import { parseConfig } from "../../src/config.ts";

Deno.test("calculator preserves rational precedence, Go operators and small number formatting", () => {
  const examples = {
    "1/3+1/6": "0.5",
    "2^3^2": "512",
    "-2^2": "-4",
    "10C3": "120",
    "5P3": "60",
    "6!": "720",
    "√9+|1-5|": "7",
    "50%": "0.5",
    "-7//3": "-2",
    "-7%3": "2",
    "7%-3": "1",
    "3.14 ×（2＋1）": "9.42",
    "1/300000": "3.333e-6",
    "1/1000000000000000000000000000000": "1e-30",
  };
  for (const [expression, expected] of Object.entries(examples)) {
    assertEquals(formatCalculation(calculate(expression)), expected, expression);
  }
  assertEquals(isCalculation("10C3"), true);
  assertEquals(isCalculation("+1"), false);
  assertEquals(isCalculation("hello=1+2"), false);
  for (
    const value of ["1/0", "(-1)^0.5", "3000!", "2^100000", "(".repeat(150) + "1", "Deno.exit()"]
  ) {
    assertThrows(() => calculate(value));
  }
});

Deno.test("exchange validates aliases, serializes concurrent fetches and rejects missing rates", async () => {
  const request = parseExchange("1.5 usd to RMB")!;
  assertEquals(request, { amount: 1.5, from: "USD", to: "CNY" });
  assertEquals(parseExchange("2 NTD"), { amount: 2, from: "TWD", to: "CNY" });
  assertEquals(parseExchange("1.4.3 usd"), undefined);
  let calls = 0;
  const fetcher: typeof fetch = () => {
    calls++;
    return Promise.resolve(
      Response.json({ result: "success", rates: { CNY: 7 }, time_last_update_unix: 123 }),
    );
  };
  const results = await Promise.all([exchange(request, fetcher), exchange(request, fetcher)]);
  assertEquals(calls, 1);
  assertEquals(results[0]!.result, 10.5);
  await assertRejects(() => exchange({ ...request, to: "XYZ" }, fetcher));
});

Deno.test("OCR uses configured API version and binary input and retains all lines", async () => {
  const config = parseConfig(
    "database-path: unused\nocr:\n  endpoint: https://ocr.test\n  api-key: fake\n  api-ver: custom\n  features: read\n  language: zh\n",
  ).ocr;
  const fetcher: typeof fetch = (input, init) => {
    const url = new URL(String(input));
    assertEquals(url.searchParams.get("api-version"), "custom");
    assertEquals(url.searchParams.get("language"), "zh");
    assertEquals(Array.from(init!.body as Uint8Array), [1, 2]);
    return Promise.resolve(
      Response.json({
        readResult: {
          blocks: [{ lines: [{ text: "一" }, { text: "二" }] }, { lines: [{ text: "三" }] }],
        },
      }),
    );
  };
  assertEquals(await ocr(new Uint8Array([1, 2]), config, fetcher), "一\n二\n\n三");
  await assertRejects(() =>
    ocr(new Uint8Array(), config, () => Promise.resolve(new Response("secret", { status: 401 })))
  );
});
