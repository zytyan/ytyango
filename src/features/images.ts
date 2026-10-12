import { DenoProcessRunner } from "../shared/process.ts";
import type { ProcessRunner } from "../shared/types.ts";
import { Semaphore } from "../shared/concurrency.ts";

const slots = new Semaphore(2);
const runner = new DenoProcessRunner();
const decoder = new TextDecoder();

async function temporary<T>(operation: (root: string) => Promise<T>): Promise<T> {
  const root = await Deno.makeTempDir({ prefix: "ytyan-image-" });
  try {
    return await slots.use(() => operation(root));
  } finally {
    await Deno.remove(root, { recursive: true });
  }
}
async function ffmpeg(process: ProcessRunner, args: string[]): Promise<void> {
  const result = await process.run("/usr/bin/ffmpeg", [
    "-hide_banner",
    "-loglevel",
    "error",
    "-nostdin",
    "-y",
    "-threads",
    "1",
    "-filter_complex_threads",
    "1",
    ...args,
  ], { timeoutMs: 30_000 });
  if (result.code !== 0) throw new Error("图片处理失败");
}
export async function normalizeModerationImage(
  bytes: Uint8Array,
  process: ProcessRunner = runner,
): Promise<Uint8Array> {
  if (bytes.length > 20 * 1024 * 1024) throw new Error("图片文件过大");
  return await temporary(async (root) => {
    await Deno.writeFile(`${root}/input`, bytes);
    const result = await process.run("/usr/bin/ffprobe", [
      "-v",
      "error",
      "-show_entries",
      "stream=width,height",
      "-of",
      "json",
      `${root}/input`,
    ], { timeoutMs: 10_000 });
    const size = (JSON.parse(decoder.decode(result.stdout)) as {
      streams?: Array<{ width: number; height: number }>;
    }).streams?.[0];
    if (
      result.code !== 0 || !size || size.width < 128 || size.height < 128 ||
      size.width * size.height > 16_000_000
    ) throw new Error("图片尺寸不符合审核要求");
    for (const quality of [5, 8, 12, 17]) {
      await ffmpeg(process, [
        "-i",
        `${root}/input`,
        "-vf",
        "scale=768:768:force_original_aspect_ratio=decrease",
        "-frames:v",
        "1",
        "-q:v",
        String(quality),
        `${root}/out.jpg`,
      ]);
      const image = await Deno.readFile(`${root}/out.jpg`);
      if (image.length <= 128 * 1024) return image;
    }
    throw new Error("审核 JPEG 超过 128 KiB");
  });
}
export async function webpToPng(bytes: Uint8Array): Promise<Uint8Array> {
  return await temporary(async (root) => {
    await Deno.writeFile(`${root}/input.webp`, bytes);
    await ffmpeg(runner, ["-i", `${root}/input.webp`, "-frames:v", "1", `${root}/out.png`]);
    return await Deno.readFile(`${root}/out.png`);
  });
}
export async function prpr(bytes: Uint8Array): Promise<Uint8Array> {
  return await temporary(async (root) => {
    await Deno.writeFile(`${root}/avatar`, bytes);
    await Deno.writeFile(
      `${root}/base.png`,
      await Deno.readFile(new URL("../assets/prpr_base.png", import.meta.url)),
    );
    await ffmpeg(runner, [
      "-i",
      `${root}/base.png`,
      "-i",
      `${root}/avatar`,
      "-filter_complex",
      "[0:v]split=2[b0][b1];[1:v]scale=143:143[avatar];[b0][avatar]overlay=306:362:format=auto[face];[face][b1]overlay=0:0:format=auto",
      "-frames:v",
      "1",
      "-c:v",
      "libwebp",
      `${root}/out.webp`,
    ]);
    return await Deno.readFile(`${root}/out.webp`);
  });
}

const aliases = [
  ["萨卡", "薩卡", "saca", "saka", "さか", "サカ"],
  ["班", "bam", "ban", "ばん", "バン"],
  ["甲", "basu", "bas", "ばす", "バス"],
  ["鱼", "魚", "pisu", "pis", "ぴす", "ピス"],
];
const segments = [[5, 235], [239, 216], [454, 235], [688, 238]] as const;
export function fishParts(text: string): number[] {
  let remaining = text.toLowerCase();
  const parts: number[] = [];
  while (remaining) {
    const match = aliases.flatMap((names, index) => names.map((name) => ({ name, index }))).find(
      (item) => remaining.startsWith(item.name),
    );
    if (!match) throw new Error("无法识别鱼图文字");
    parts.push(match.index);
    remaining = remaining.slice(match.name.length);
    if (parts.length > 100) throw new Error("最多拼接 100 个鱼图部件");
  }
  if (!parts.length) throw new Error("没有鱼图文字");
  return parts;
}
export function isFish(text: string): boolean {
  return /^(?:(?:萨卡|薩卡|saca|saka|さか|サカ|班|bam|ban|ばん|バン|甲|basu?|ばす|バス|鱼|魚|pisu?|ぴす|ピス))+$/i
    .test(text);
}
export async function fish(text: string): Promise<Uint8Array> {
  const parts = fishParts(text);
  const positions = [50];
  for (let i = 1; i < parts.length; i++) {
    const previous = parts[i - 1]!;
    const current = parts[i]!;
    const overlap = current === 0
      ? (previous === 0 ? 50 : previous === 3 ? 80 : 70)
      : previous === 3 && current === 3
      ? 20
      : 0;
    positions.push(positions[i - 1]! + segments[previous]![1] - overlap);
  }
  const width = positions.at(-1)! + segments[parts.at(-1)!]![1] + 50;
  return await temporary(async (root) => {
    await Deno.writeFile(
      `${root}/base.png`,
      await Deno.readFile(new URL("../assets/sakabanbasupisu.png", import.meta.url)),
    );
    const split = parts.length > 1
      ? `[0:v]split=${parts.length}${parts.map((_p, i) => `[s${i}]`).join("")};`
      : "";
    const filters = parts.map((part, i) =>
      `${parts.length > 1 ? `[s${i}]` : "[0:v]"}crop=${segments[part]![1]}:371:${
        segments[part]![0]
      }:0[p${i}];`
    );
    filters.push(`color=c=0xfe8f00:s=${width}x371[bg];`);
    parts.forEach((_part, i) =>
      filters.push(
        `${i === 0 ? "[bg]" : `[o${i - 1}]`}[p${i}]overlay=${positions[i]}:0${
          i === parts.length - 1 ? "[out]" : `[o${i}]`
        }${i === parts.length - 1 ? "" : ";"}`,
      )
    );
    await ffmpeg(runner, [
      "-i",
      `${root}/base.png`,
      "-filter_complex",
      split + filters.join(""),
      "-map",
      "[out]",
      "-frames:v",
      "1",
      `${root}/out.jpg`,
    ]);
    return await Deno.readFile(`${root}/out.jpg`);
  });
}
