import type { AppConfig } from "../config.ts";
import { BoundedCache, KeyedMutex } from "../shared/concurrency.ts";

const aliases: Record<string, string> = { RMB: "CNY", NTD: "TWD", SKW: "KRW" };
export function parseExchange(text: string) {
  const match = /^(\d+(?:\.\d+)?)\s*([a-z]{3})\s*(?:(?:to)?\s*([a-z]{3}))?$/i.exec(text.trim());
  if (!match) return undefined;
  const currency = (value: string) => aliases[value.toUpperCase()] ?? value.toUpperCase();
  return { amount: Number(match[1]), from: currency(match[2]!), to: currency(match[3] ?? "CNY") };
}
const rateCache = new BoundedCache<string, { rates: Record<string, number>; updated: number }>(
  20,
  3600_000,
);
const locks = new KeyedMutex();
export async function exchange(
  request: NonNullable<ReturnType<typeof parseExchange>>,
  fetcher: typeof fetch = fetch,
) {
  return await locks.use(request.from, async () => {
    if (request.from === request.to) {
      return { ...request, result: request.amount, updated: Math.floor(Date.now() / 1000) };
    }
    let cached = rateCache.get(request.from);
    if (!cached) {
      const response = await fetcher(
        `https://open.er-api.com/v6/latest/${encodeURIComponent(request.from)}`,
        { signal: AbortSignal.timeout(10_000) },
      );
      if (!response.ok) throw new Error("汇率服务暂时不可用");
      const body = await response.json() as {
        result: string;
        rates: Record<string, number>;
        time_last_update_unix: number;
      };
      if (
        body.result !== "success" || !body.rates || !Number.isFinite(body.time_last_update_unix)
      ) throw new Error("汇率响应无效");
      cached = { rates: body.rates, updated: body.time_last_update_unix };
      rateCache.set(request.from, cached);
    }
    const rate = cached.rates[request.to];
    if (rate === undefined || !Number.isFinite(rate) || rate <= 0) throw new Error("不支持该币种");
    const result = request.amount * rate;
    if (!Number.isFinite(result)) throw new Error("金额过大");
    return { ...request, result, updated: cached.updated };
  });
}
export async function ocr(
  bytes: Uint8Array,
  config: AppConfig["ocr"],
  fetcher: typeof fetch = fetch,
): Promise<string> {
  if (!config.endpoint || !config.apiKey) throw new Error("OCR 未配置");
  const url = new URL(`${config.endpoint}/computervision/imageanalysis:analyze`);
  url.searchParams.set("api-version", config.apiVersion);
  if (config.features) url.searchParams.set("features", config.features);
  if (config.language) url.searchParams.set("language", config.language);
  const response = await fetcher(url, {
    method: "POST",
    headers: { "content-type": "image/jpeg", "Ocp-Apim-Subscription-Key": config.apiKey },
    body: Uint8Array.from(bytes),
    signal: AbortSignal.timeout(10_000),
  });
  if (!response.ok) throw new Error(`OCR HTTP ${response.status}`);
  const body = await response.json() as {
    error?: { code?: string };
    readResult?: { blocks?: Array<{ lines?: Array<{ text: string }> }> };
  };
  if (body.error?.code) throw new Error("OCR 响应失败");
  return body.readResult?.blocks?.map((block) =>
    block.lines?.map((line) => line.text).join("\n") ?? ""
  ).join("\n\n") ?? "";
}
