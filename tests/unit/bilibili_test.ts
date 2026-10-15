import { assert, assertEquals } from "@std/assert";
import { convertBilibiliLinks, hasBilibiliVideo } from "../../src/download/bilibili.ts";

Deno.test("Bilibili links are cleaned and retain both AV/BV forms", async () => {
  const result = await convertBilibiliLinks(
    "看 https://www.bilibili.com/video/BV1xx411c7mD?spm_id_from=333.1.0.0&t=12",
  );
  assert(result.needClean);
  assert(result.hasAv);
  assert(result.hasBv);
  assert(!result.bvText.includes("spm_id_from"));
  assert(result.bvText.includes("t=12"));
  assertEquals(hasBilibiliVideo(result.bvText), true);
});

Deno.test("short links count as conversions without tracking parameters and follow bounded relative hops", async () => {
  const calls: string[] = [];
  let cancelled = 0;
  const fetcher: typeof fetch = (input) => {
    calls.push(String(input));
    return Promise.resolve(
      new Response(
        new ReadableStream({
          cancel() {
            cancelled++;
          },
        }),
        {
          status: 302,
          headers: {
            location: calls.length === 1
              ? "/second"
              : "https://www.bilibili.com/video/BV17x411w7KC?p=1",
          },
        },
      ),
    );
  };
  const result = await convertBilibiliLinks("看 https://b23.tv/first", fetcher);
  assertEquals(calls, ["https://b23.tv/first", "https://b23.tv/second"]);
  assertEquals(result.needClean, true);
  assertEquals(result.bvText, "看 https://www.bilibili.com/video/BV17x411w7KC?p=1");
  assertEquals(cancelled, 2);
});

Deno.test("expired short links report Bilibili JSON errors and reject loops or external redirect destinations", async () => {
  const { BilibiliLinkError } = await import("../../src/download/bilibili.ts");
  const { assertRejects } = await import("@std/assert");
  await assertRejects(
    () =>
      convertBilibiliLinks(
        "https://b23.tv/YpKSaml",
        () => Promise.resolve(Response.json({ code: -404, message: "private upstream data" })),
      ),
    BilibiliLinkError,
    "上游返回 -404",
  );
  let calls = 0;
  await assertRejects(
    () =>
      convertBilibiliLinks("https://b23.tv/loop", () => {
        calls++;
        return Promise.resolve(new Response(null, { status: 302, headers: { location: "/loop" } }));
      }),
    BilibiliLinkError,
    "循环跳转",
  );
  assertEquals(calls, 1);
  await assertRejects(
    () =>
      convertBilibiliLinks("https://b23.tv/external", () =>
        Promise.resolve(
          new Response(null, { status: 302, headers: { location: "http://127.0.0.1/private" } }),
        )),
    BilibiliLinkError,
    "非B站跳转地址",
  );
});
