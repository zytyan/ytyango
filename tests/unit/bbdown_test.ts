import { assertEquals, assertThrows } from "@std/assert";
import { bbdownArgs, bbdownOutput } from "../../src/download/service.ts";

Deno.test("bbdown-go selects audio output and rejects missing or escaping paths", () => {
  assertEquals(
    bbdownArgs("/account.json", "/tmp/download", true, "https://bilibili.com/video/BV1"),
    [
      "--cookie-file",
      "/account.json",
      "-work-dir",
      "/tmp/download",
      "-encoding-priority",
      "HEVC",
      "--audio-only",
      "https://bilibili.com/video/BV1",
    ],
  );
  const bytes = (body: unknown) => new TextEncoder().encode(JSON.stringify(body));
  assertEquals(
    bbdownOutput(bytes({ video_path: "v.mp4", audio_path: "a.m4a" }), "/tmp/download", true),
    "/tmp/download/a.m4a",
  );
  assertThrows(() => bbdownOutput(bytes({ video_path: "v.mp4" }), "/tmp/download", true));
  assertThrows(() => bbdownOutput(bytes({ video_path: "../outside.mp4" }), "/tmp/download", false));
  assertThrows(() => bbdownOutput(bytes({ video_path: "/etc/passwd" }), "/tmp/download", false));
});
