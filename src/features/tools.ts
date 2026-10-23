import type { AppConfig } from "../config.ts";
import { BoundedCache, KeyedMutex } from "../shared/concurrency.ts";

const aliases: Record<string, string> = { RMB: "CNY", NTD: "TWD", SKW: "KRW" };
export function parseExchange(text: string) {
  const value = text.trim();
  const match = /^(\d+(?:\.\d+)?)\s*([a-z]{3})\s*(?:(?:to)?\s*([a-z]{3}))?$/i.exec(value) ??
    /^(\d+(?:\.\d+)?)\s*([a-z0-9]{2,12})(?:\s+(?:to\s+)?([a-z0-9]{2,12}))?$/i.exec(value);
  if (!match || !/[a-z]/i.test(match[2]!) || (match[3] && !/[a-z]/i.test(match[3]))) {
    return undefined;
  }
  const amount = Number(match[1]);
  if (!Number.isFinite(amount)) return undefined;
  const currency = (value: string) => aliases[value.toUpperCase()] ?? value.toUpperCase();
  return { amount, from: currency(match[2]!), to: currency(match[3] ?? "CNY") };
}

export type ExchangeUpdate = { precision: "day"; date: string } | {
  precision: "second";
  unix: number;
};
const localTime = (milliseconds: number) =>
  new Date(milliseconds + 8 * 3600_000).toISOString().slice(0, 19).replace("T", " ");
export function formatExchangeUpdate(updated: ExchangeUpdate | undefined): string {
  if (!updated) return "";
  if (updated.precision === "second") {
    return `汇率更新于：${localTime(updated.unix * 1000)}（UTC+8）`;
  }
  const start = Date.parse(`${updated.date}T00:00:00Z`);
  return `汇率更新于：${localTime(start)} 至 ${
    localTime(start + 86400_000)
  } 前（UTC+8，来源仅精确到日）`;
}

interface ExchangeRates {
  rates: Record<string, number>;
  updated: ExchangeUpdate;
}
const rateCache = new BoundedCache<string, ExchangeRates>(20, 3600_000);
const locks = new KeyedMutex();
const sources = ["cdn", "cloudflare", "legacy"] as const;
export class UnsupportedCurrencyError extends Error {
  constructor() {
    super("不支持该币种");
    this.name = "UnsupportedCurrencyError";
  }
}
async function exchangeRates(
  source: typeof sources[number],
  from: string,
  fetcher: typeof fetch,
): Promise<ExchangeRates> {
  const key = `${source}:${from}`;
  return await locks.use(key, async () => {
    const cached = rateCache.get(key);
    if (cached) return cached;
    const code = from.toLowerCase();
    const url = source === "cdn"
      ? `https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/${code}.min.json`
      : source === "cloudflare"
      ? `https://latest.currency-api.pages.dev/v1/currencies/${code}.min.json`
      : `https://open.er-api.com/v6/latest/${from}`;
    const response = await fetcher(url, { signal: AbortSignal.timeout(10_000) });
    if (!response.ok) {
      await response.body?.cancel();
      if (response.status === 404) throw new UnsupportedCurrencyError();
      throw new Error("汇率服务暂时不可用");
    }
    const body = await response.json() as Record<string, unknown>;
    let updated: ExchangeUpdate;
    if (source === "legacy") {
      if (body.result === "error" && body["error-type"] === "unsupported-code") {
        throw new UnsupportedCurrencyError();
      }
      const unix = body.time_last_update_unix;
      if (
        body.result !== "success" || typeof unix !== "number" || unix <= 0 ||
        !Number.isFinite(unix) || !Number.isFinite(new Date(unix * 1000).getTime())
      ) throw new Error("汇率更新时间无效");
      updated = { precision: "second", unix };
    } else {
      const date = body.date;
      if (
        typeof date !== "string" || !/^\d{4}-\d{2}-\d{2}$/.test(date) ||
        !Number.isFinite(Date.parse(`${date}T00:00:00Z`)) ||
        new Date(`${date}T00:00:00Z`).toISOString().slice(0, 10) !== date
      ) throw new Error("汇率更新日期无效");
      updated = { precision: "day", date };
    }
    const raw = source === "legacy" ? body.rates : body[code];
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) throw new Error("汇率响应无效");
    const rates: Record<string, number> = {};
    for (const [currency, rate] of Object.entries(raw)) {
      if (typeof rate === "number" && Number.isFinite(rate) && rate > 0) {
        rates[currency.toUpperCase()] = rate;
      }
    }
    if (!Object.keys(rates).length) throw new Error("汇率响应无效");
    const result = { rates, updated };
    rateCache.set(key, result);
    return result;
  });
}
export async function exchange(
  request: NonNullable<ReturnType<typeof parseExchange>>,
  fetcher: typeof fetch = fetch,
) {
  if (
    !Number.isFinite(request.amount) ||
    !/^[A-Z0-9]{2,12}$/.test(request.from) || !/^[A-Z0-9]{2,12}$/.test(request.to)
  ) throw new Error("汇率请求无效");
  if (request.from === request.to) {
    return { ...request, result: request.amount, updated: undefined };
  }
  let unsupported = false;
  for (const source of sources) {
    let data: ExchangeRates;
    try {
      data = await exchangeRates(source, request.from, fetcher);
    } catch (error) {
      if (error instanceof UnsupportedCurrencyError) unsupported = true;
      continue;
    }
    const rate = data.rates[request.to];
    if (rate === undefined) {
      unsupported = true;
      continue;
    }
    const result = request.amount * rate;
    if (!Number.isFinite(result)) throw new Error("金额过大");
    return { ...request, result, updated: data.updated };
  }
  if (unsupported) throw new UnsupportedCurrencyError();
  throw new Error("汇率服务暂时不可用");
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
