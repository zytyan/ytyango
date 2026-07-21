const TABLE = "FcwAPNKTMug3GV5Lj7EJnHpWsx4tb8haYeviqBz6rkCy12mUSDQX9RdoZf";
const XOR_CODE = 23442827791579n;
const BASE = 58n;
const LINK = /(?:(?:https?:\/\/)?\w+(?:\.\w+)+\/[\w#?/=&%+._()*$@!^\-]+)/gi;
const VIDEO = /\/(av\d+|BV[\da-zA-Z]+)/i;

function avToBv(path: string): string {
  const match = path.match(/^\/av(\d+)$/i);
  if (!match) return path;
  let value = (BigInt(match[1]!) | (1n << 51n)) ^ XOR_CODE;
  const chars = Array.from("BV1000000000");
  for (let index = chars.length - 1; value > 0n; index--) {
    chars[index] = TABLE[Number(value % BASE)]!;
    value /= BASE;
  }
  [chars[3], chars[9]] = [chars[9]!, chars[3]!];
  [chars[4], chars[7]] = [chars[7]!, chars[4]!];
  return `/${chars.join("")}`;
}

function bvToAv(path: string): string {
  if (path.length !== 13 || !path.startsWith("/BV1")) return path;
  const chars = Array.from(path.slice(1));
  [chars[3], chars[9]] = [chars[9]!, chars[3]!];
  [chars[4], chars[7]] = [chars[7]!, chars[4]!];
  let value = 0n;
  for (const character of chars.slice(3)) {
    const index = TABLE.indexOf(character);
    if (index < 0) return path;
    value = value * BASE + BigInt(index);
  }
  return `/av${(value & ((1n << 51n) - 1n)) ^ XOR_CODE}`;
}

export interface ConvertedBilibiliLinks {
  readonly raw: string;
  readonly avText: string;
  readonly bvText: string;
  readonly hasAv: boolean;
  readonly hasBv: boolean;
  readonly needClean: boolean;
}

async function redirectLocation(url: string, fetcher: typeof fetch): Promise<string | undefined> {
  const response = await fetcher(url, { redirect: "manual", signal: AbortSignal.timeout(10_000) });
  return response.headers.get("location") ?? undefined;
}

async function convertOne(raw: string, fetcher: typeof fetch): Promise<{
  av: string;
  bv: string;
  hasAv: boolean;
  hasBv: boolean;
  cleaned: boolean;
}> {
  let url: URL;
  try {
    url = new URL(/^https?:\/\//i.test(raw) ? raw : `https://${raw}`);
  } catch {
    return { av: raw, bv: raw, hasAv: false, hasBv: false, cleaned: false };
  }
  url.protocol = "https:";
  if (["b23.tv", "bili2233.cn"].includes(url.hostname)) {
    const location = await redirectLocation(url.href, fetcher);
    if (!location) return { av: raw, bv: raw, hasAv: false, hasBv: false, cleaned: false };
    url = new URL(location);
  }
  const allowed = url.hostname === "mall.bilibili.com"
    ? new Set(["itemsId"])
    : new Set(["p", "start_progress", "t"]);
  let cleaned = false;
  for (const key of [...url.searchParams.keys()]) {
    if (!allowed.has(key)) {
      url.searchParams.delete(key);
      cleaned = true;
    }
  }
  if (url.hostname === "mall.bilibili.com") {
    return { av: url.href, bv: url.href, hasAv: false, hasBv: true, cleaned };
  }
  if (!["bilibili.com", "www.bilibili.com", "m.bilibili.com"].includes(url.hostname)) {
    return { av: raw, bv: raw, hasAv: false, hasBv: false, cleaned: false };
  }
  const match = url.pathname.match(VIDEO);
  if (!match) return { av: url.href, bv: url.href, hasAv: false, hasBv: true, cleaned };
  const token = `/${match[1]}`;
  if (/^\/BV/i.test(token)) {
    return {
      av: url.href.replace(token, bvToAv(token)),
      bv: url.href,
      hasAv: true,
      hasBv: true,
      cleaned,
    };
  }
  return {
    av: url.href,
    bv: url.href.replace(token, avToBv(token)),
    hasAv: true,
    hasBv: true,
    cleaned,
  };
}

export async function convertBilibiliLinks(
  text: string,
  fetcher: typeof fetch = fetch,
): Promise<ConvertedBilibiliLinks> {
  if (!/(?:bilibili\.com|b23\.tv|bili2233\.cn)/i.test(text)) throw new Error("no Bilibili links");
  const matches = [...text.matchAll(LINK)];
  if (matches.length === 0) throw new Error("no Bilibili links");
  let av = "";
  let bv = "";
  let cursor = 0;
  let hasAv = false;
  let hasBv = false;
  let needClean = false;
  for (const match of matches) {
    const start = match.index ?? 0;
    av += text.slice(cursor, start);
    bv += text.slice(cursor, start);
    const converted = await convertOne(match[0], fetcher);
    av += converted.av;
    bv += converted.bv;
    hasAv ||= converted.hasAv;
    hasBv ||= converted.hasBv;
    needClean ||= converted.cleaned;
    cursor = start + match[0].length;
  }
  av += text.slice(cursor);
  bv += text.slice(cursor);
  return { raw: text, avText: av, bvText: bv, hasAv, hasBv, needClean };
}

export function hasBilibiliVideo(text: string): boolean {
  return VIDEO.test(text);
}
