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
