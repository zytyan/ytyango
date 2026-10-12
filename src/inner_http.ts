import { timingSafeEqual } from "node:crypto";
import { Session } from "node:inspector/promises";
import type { MainDatabase } from "./db/database.ts";
import type { StatsService } from "./stats/service.ts";
import { createBackup } from "./backup.ts";
import { Semaphore } from "./shared/concurrency.ts";

export function innerAddress(value = "", token = "") {
  if (value.trim().toUpperCase() === "OFF") return undefined;
  const url = new URL(`http://${value.trim() || "127.0.0.1:4019"}`);
  const host = url.hostname.replace(/^\[|\]$/g, "");
  if (url.pathname !== "/" || url.username || url.password || !url.port) {
    throw new Error("invalid BOT_INNER_HTTP");
  }
  if (!["localhost", "::1"].includes(host) && !/^127\.\d+\.\d+\.\d+$/.test(host) && !token.trim()) {
    throw new Error("non-loopback inner HTTP requires GOYTYAN_BACKUP_TOKEN");
  }
  return { hostname: host, port: Number(url.port) };
}

export function installLogging(initial: number) {
  let level = initial;
  const original = {
    debug: console.debug,
    log: console.log,
    info: console.info,
    warn: console.warn,
    error: console.error,
  };
  for (
    const [name, severity] of [["debug", -4], ["log", 0], ["info", 0], ["warn", 4], [
      "error",
      8,
    ]] as const
  ) {
    console[name] = (...args: unknown[]) => {
      if (severity >= level) original[name](...args);
    };
  }
  return {
    get: () => level,
    set: (value: number) => {
      level = value;
    },
    restore: () => Object.assign(console, original),
  };
}

function authorized(request: Request, token: string): boolean {
  if (!token) return true;
  const actual = request.headers.get("X-Backup-Token") ??
    new URL(request.url).searchParams.get("token") ?? "";
  const encoder = new TextEncoder(), a = encoder.encode(actual), b = encoder.encode(token);
  return a.length === b.length && timingSafeEqual(a, b);
}

async function body(request: Request): Promise<Record<string, unknown>> {
  if (!request.body) throw new Error("missing body");
  const reader = request.body.getReader(), parts: Uint8Array[] = [];
  let size = 0;
  try {
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      size += value.length;
      if (size > 1_048_576) {
        await reader.cancel();
        throw new Error("body too large");
      }
      parts.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  const bytes = new Uint8Array(size);
  let offset = 0;
  for (const part of parts) {
    bytes.set(part, offset);
    offset += part.length;
  }
  const decoded = JSON.parse(new TextDecoder().decode(bytes));
  if (!decoded || typeof decoded !== "object" || Array.isArray(decoded)) {
    throw new Error("invalid body");
  }
  return decoded;
}

export function innerHandler(options: {
  db: MainDatabase;
  stats: StatsService;
  mediaRoot: string;
  token: string;
  signal: AbortSignal;
  backupDurationMs?: number;
  logging: { get: () => number; set: (value: number) => void };
}) {
  const backupSlot = new Semaphore(1), profileSlot = new Semaphore(1);
  return async (request: Request): Promise<Response> => {
    if (!authorized(request, options.token)) return new Response("unauthorized", { status: 401 });
    const url = new URL(request.url), method = request.method;
    const signal = AbortSignal.any([request.signal, options.signal]);
    try {
      if (method === "POST" && ["/mars-counter", "/dio-ban"].includes(url.pathname)) {
        let value: Record<string, unknown>;
        try {
          value = await body(request);
        } catch {
          return new Response("invalid JSON body", { status: 400 });
        }
        const group = value.group_id;
        if (!Number.isSafeInteger(group)) return new Response("invalid group_id", { status: 400 });
        if (url.pathname === "/mars-counter") {
          if (!Number.isSafeInteger(value.mars_count) || Number(value.mars_count) < 0) {
            return new Response("invalid mars_count", { status: 400 });
          }
          options.stats.external(group as number, "mars", Number(value.mars_count));
        } else {
          if (
            !Number.isSafeInteger(value.user_id) || ![0, 1, 2, 3].includes(Number(value.action))
          ) return new Response("invalid dio action", { status: 400 });
          options.stats.external(group as number, value.action === 0 ? "dio_add" : "dio_ban");
        }
        return new Response(null, { status: 200 });
      }
      if (method === "GET" && url.pathname === "/loggers") {
        return new Response(`runtime\t${options.logging.get()}\n`);
      }
      if (method === "PUT" && url.pathname.startsWith("/loggers/")) {
        const match = /^\/loggers\/runtime\/(-?\d+)$/.exec(url.pathname),
          level = Number(match?.[1]);
        if (!match || level < -128 || level > 127) {
          return new Response("invalid runtime log level", { status: 400 });
        }
        options.logging.set(level);
        return new Response(null, { status: 204 });
      }
      if (method === "GET" && url.pathname === "/debug/runtime") {
        return Response.json({
          runtime: "Deno",
          version: Deno.version,
          memory: Deno.memoryUsage(),
          uptime_seconds: performance.now() / 1000,
        });
      }
      if (method === "GET" && ["/debug/profile", "/debug/pprof/profile"].includes(url.pathname)) {
        const seconds = Number(url.searchParams.get("seconds") ?? 30);
        if (!Number.isInteger(seconds) || seconds < 1 || seconds > 60) {
          return new Response("seconds must be 1..60", { status: 400 });
        }
        return await profileSlot.use(async () => {
          signal.throwIfAborted();
          const session = new Session();
          try {
            session.connect();
            await session.post("Profiler.enable");
            await session.post("Profiler.start");
            await new Promise<void>((resolve, reject) => {
              const timer = setTimeout(() => {
                signal.removeEventListener("abort", abort);
                resolve();
              }, seconds * 1000);
              const abort = () => {
                clearTimeout(timer);
                reject(signal.reason);
              };
              signal.addEventListener("abort", abort, { once: true });
              if (signal.aborted) abort();
            });
            const { profile } = await session.post("Profiler.stop");
            return Response.json(profile, {
              headers: { "content-disposition": "attachment; filename=runtime.cpuprofile" },
            });
          } finally {
            session.disconnect();
          }
        });
      }
      if (method === "GET" && url.pathname === "/backupdb") {
        const selection = (url.searchParams.get("db") ?? "main").toLowerCase();
        if (selection === "msg") {
          return new Response("legacy message database retired", { status: 410 });
        }
        const media = (url.searchParams.get("media") ?? "0").toLowerCase();
        if (!["main", "all"].includes(selection) || !["0", "1", "true", "false"].includes(media)) {
          return new Response("invalid backup selection", { status: 400 });
        }
        const includeMedia = ["1", "true"].includes(media);
        const deadline = AbortSignal.any([
          signal,
          AbortSignal.timeout(includeMedia ? options.backupDurationMs ?? 30 * 60_000 : 30_000),
        ]);
        return await backupSlot.use(async () => {
          deadline.throwIfAborted();
          options.stats.flush();
          const result = await createBackup(options.db, options.mediaRoot, includeMedia, deadline);
          const file = await Deno.open(result.path, { read: true });
          const reader = file.readable.getReader();
          let cleaned = false;
          const cleanup = async () => {
            if (!cleaned) {
              cleaned = true;
              try {
                await reader.cancel();
              } finally {
                await result.cleanup();
              }
            }
          };
          const stream = new ReadableStream<Uint8Array>({
            async pull(controller) {
              try {
                deadline.throwIfAborted();
                const { done, value } = await reader.read();
                if (done) {
                  await cleanup();
                  controller.close();
                } else controller.enqueue(value);
              } catch (error) {
                await cleanup();
                controller.error(error);
              }
            },
            cancel: cleanup,
          });
          return new Response(stream, {
            headers: {
              "content-type": "application/gzip",
              "content-disposition": "attachment; filename=ytyan-backup.tar.gz",
            },
          });
        });
      }
      if (method === "GET" && url.pathname === "/") {
        return new Response(
          "POST /mars-counter\nPOST /dio-ban\nGET /backupdb?db=main|all&media=0|1\nGET /loggers\nPUT /loggers/runtime/<level>\nGET /debug/runtime\nGET /debug/profile?seconds=1..60\n",
        );
      }
      return new Response("not found", { status: 404 });
    } catch (error) {
      console.error("inner HTTP request failed", url.pathname, error);
      return new Response("request failed", { status: 500 });
    }
  };
}

export function startInnerHttp(
  db: MainDatabase,
  stats: StatsService,
  mediaRoot: string,
  signal: AbortSignal,
  logging: { get: () => number; set: (value: number) => void },
) {
  const token = Deno.env.get("GOYTYAN_BACKUP_TOKEN") ?? "";
  const address = innerAddress(Deno.env.get("BOT_INNER_HTTP"), token);
  if (!address) return undefined;
  const raw = Deno.env.get("GOYTYAN_BACKUP_MAX_DURATION") ?? "30m";
  const duration = /^(\d+)(ms|s|m|h)$/.exec(raw);
  if (!duration || Number(duration[1]) === 0) {
    throw new Error("invalid GOYTYAN_BACKUP_MAX_DURATION");
  }
  const factors: Record<string, number> = { ms: 1, s: 1000, m: 60_000, h: 3600_000 };
  const backupDurationMs = Number(duration[1]) * factors[duration[2]!]!;
  return Deno.serve(
    { ...address, signal },
    innerHandler({ db, stats, mediaRoot, signal, token, logging, backupDurationMs }),
  );
}
