import { cpuUsage } from "node:process";
import { AsyncLocalStorage } from "node:async_hooks";
import { logger } from "../logging.ts";

interface Timing {
  calls: number;
  total_ms: number;
  max_ms: number;
  failures: number;
  first_start_ms: number;
  last_end_ms: number;
}
interface Measurements {
  history_messages?: number;
  history_media_files?: number;
  history_media_bytes?: number;
  history_assistant_payload_bytes?: number;
  context_snapshot_chars?: number;
  request_body_chars?: number;
  response_body_bytes?: number;
  assistant_payload_bytes?: number;
}
interface SyncSpan {
  stage: string;
  started_at: string;
  start_ms: number;
  end_ms: number;
  elapsed_ms: number;
  update_id?: number;
  run_id?: number;
  failed: boolean;
}
interface Trace {
  update_id: number;
  kind: string;
  started_at: string;
  start: number;
  received?: number;
  transport_lag_ms?: number;
  runner_queue_ms?: number;
  tags: {
    session_id?: number;
    run_id?: number;
    provider?: string;
    model?: string;
    response_model?: string;
  };
  measurements: Measurements;
  stages: Map<string, Timing>;
  running: Map<number, { stage: string; start: number }>;
  next: number;
}
const scope = new AsyncLocalStorage<Trace>();
const active = new Map<number, Trace>();
const recent: ReturnType<typeof summary>[] = [];
const received = new Map<number, { monotonic: number; wall: number }>();
let loopLagMs = 0;
let previousLoopLagMs = 0;
const background = new Map<string, { calls: number; last_ms: number; max_ms: number }>();
const slowSyncSpans: SyncSpan[] = [];
const loopStalls: {
  observed_at: string;
  lag_ms: number;
  sample: ReturnType<typeof resourceSample>;
  synchronous_spans: SyncSpan[];
  requests: ReturnType<typeof summary>[];
}[] = [];
let runtimeSample: ReturnType<typeof resourceSample> | undefined;
const rounded = (n: number) => Math.round(n * 10) / 10;

/** Only IDs/times are retained until dispatch, never update bodies or chat history. */
export function updatesReceived(ids: readonly number[]): void {
  const time = { monotonic: performance.now(), wall: Date.now() };
  for (const id of ids) {
    if (!received.has(id)) received.set(id, time);
  }
  while (received.size > 2048) received.delete(received.keys().next().value!);
}

export function forgetUpdate(id: number): void {
  received.delete(id);
}

export async function traceUpdate<T>(
  updateId: number,
  kind: string,
  sentAt: number | undefined,
  operation: () => Promise<T>,
): Promise<T> {
  const arrival = received.get(updateId);
  received.delete(updateId);
  const start = performance.now();
  const trace: Trace = {
    update_id: updateId,
    kind,
    started_at: new Date().toISOString(),
    start,
    received: arrival?.monotonic,
    transport_lag_ms: sentAt === undefined
      ? undefined
      : (arrival?.wall ?? Date.now()) - sentAt * 1000,
    runner_queue_ms: arrival ? rounded(start - arrival.monotonic) : undefined,
    tags: {},
    measurements: {},
    stages: new Map(),
    running: new Map(),
    next: 0,
  };
  active.set(updateId, trace);
  let failed = false;
  try {
    return await scope.run(trace, operation);
  } catch (error) {
    failed = true;
    throw error;
  } finally {
    active.delete(updateId);
    const result = { ...summary(trace), failed };
    recent.push(result);
    if (recent.length > 100) recent.shift();
    const slow = result.elapsed_ms >= 5000 || (result.transport_lag_ms ?? 0) >= 10000;
    logger("runtime")[slow ? "warn" : "info"]("latency", JSON.stringify(result));
  }
}

export function traceTags(tags: Trace["tags"]): void {
  const trace = scope.getStore();
  if (trace) Object.assign(trace.tags, tags);
}

/** Fixed numeric fields only: never pass text, payloads, URLs or identifiers as metric names. */
export function traceMeasurements(values: Measurements): void {
  const trace = scope.getStore();
  if (!trace) return;
  for (const [key, value] of Object.entries(values)) {
    if (typeof value === "number" && Number.isFinite(value) && value >= 0) {
      Object.assign(trace.measurements, { [key]: value });
    }
  }
}

function beginStage(name: string): ((failed: boolean) => void) | undefined {
  const trace = scope.getStore();
  if (!trace) return undefined;
  const id = trace.next++, start = performance.now();
  trace.running.set(id, { stage: name, start });
  return (failed) => {
    trace.running.delete(id);
    const end = performance.now();
    const elapsed = end - start;
    const previous = trace.stages.get(name) ?? {
      calls: 0,
      total_ms: 0,
      max_ms: 0,
      failures: 0,
      first_start_ms: start - trace.start,
      last_end_ms: 0,
    };
    previous.calls++;
    previous.total_ms += elapsed;
    previous.max_ms = Math.max(previous.max_ms, elapsed);
    previous.failures += Number(failed);
    previous.last_end_ms = Math.max(previous.last_end_ms, end - trace.start);
    trace.stages.set(name, previous);
  };
}

export async function latencyStage<T>(
  name: string,
  operation: () => Promise<T>,
  failedResult?: (value: T) => boolean,
): Promise<T> {
  const finish = beginStage(name);
  let failed = false;
  try {
    const result = await operation();
    failed = failedResult?.(result) ?? false;
    return result;
  } catch (error) {
    failed = true;
    throw error;
  } finally {
    finish?.(failed);
  }
}

export function latencySyncStage<T>(name: string, operation: () => T): T {
  const start = performance.now();
  const trace = scope.getStore();
  const finish = beginStage(name);
  let failed = false;
  try {
    return operation();
  } catch (error) {
    failed = true;
    throw error;
  } finally {
    finish?.(failed);
    const end = performance.now();
    if (end - start >= 50) {
      slowSyncSpans.push({
        stage: name,
        started_at: new Date(Date.now() - (end - start)).toISOString(),
        start_ms: rounded(start),
        end_ms: rounded(end),
        elapsed_ms: rounded(end - start),
        update_id: trace?.update_id,
        run_id: trace?.tags.run_id,
        failed,
      });
      if (slowSyncSpans.length > 64) slowSyncSpans.shift();
    }
  }
}

function summary(trace: Trace) {
  const now = performance.now();
  return {
    update_id: trace.update_id,
    kind: trace.kind,
    started_at: trace.started_at,
    ...trace.tags,
    elapsed_ms: rounded(now - trace.start),
    since_received_ms: trace.received === undefined ? undefined : rounded(now - trace.received),
    transport_lag_ms: trace.transport_lag_ms,
    runner_queue_ms: trace.runner_queue_ms,
    measurements: { ...trace.measurements },
    stages: Object.fromEntries([...trace.stages].map(([name, value]) => [name, {
      ...value,
      total_ms: rounded(value.total_ms),
      max_ms: rounded(value.max_ms),
      first_start_ms: rounded(value.first_start_ms),
      last_end_ms: rounded(value.last_end_ms),
    }])),
    waiting: [...trace.running.values()].map((value) => ({
      stage: value.stage,
      elapsed_ms: rounded(now - value.start),
    })),
  };
}

export function latencySnapshot() {
  return {
    active_count: active.size,
    pending_dispatch_count: received.size,
    oldest_pending_dispatch_ms: rounded(
      Math.max(0, ...[...received.values()].map((t) => performance.now() - t.monotonic)),
    ),
    event_loop_recent_max_lag_ms: rounded(Math.max(loopLagMs, previousLoopLagMs)),
    runtime_sample: runtimeSample,
    slow_synchronous_spans: [...slowSyncSpans],
    event_loop_stalls: [...loopStalls],
    background: Object.fromEntries(background),
    active: [...active.values()].map(summary),
    recent: [...recent],
  };
}

export function backgroundTiming<T>(name: string, operation: () => T): T {
  const start = performance.now();
  try {
    return operation();
  } finally {
    const elapsed = rounded(performance.now() - start);
    const previous = background.get(name) ?? { calls: 0, last_ms: 0, max_ms: 0 };
    if (!background.has(name) && background.size >= 256) {
      background.delete(background.keys().next().value!);
    }
    background.set(name, {
      calls: previous.calls + 1,
      last_ms: elapsed,
      max_ms: Math.max(previous.max_ms, elapsed),
    });
  }
}

function resourceSample(previousCpu: ReturnType<typeof cpuUsage>, elapsedMs: number) {
  const cpu = cpuUsage(previousCpu);
  const memory = Deno.memoryUsage();
  return {
    sampled_at: new Date().toISOString(),
    interval_ms: rounded(elapsedMs),
    cpu_user_ms: rounded(cpu.user / 1000),
    cpu_system_ms: rounded(cpu.system / 1000),
    // Process CPU includes runtime threads and may exceed one core (100%).
    cpu_percent: rounded((cpu.user + cpu.system) / (Math.max(1, elapsedMs) * 10)),
    rss_bytes: memory.rss,
    heap_used_bytes: memory.heapUsed,
    heap_total_bytes: memory.heapTotal,
    external_bytes: memory.external,
  };
}

export function startLatencyMonitor(intervalMs = 1000, stallThresholdMs = 500): () => void {
  let lastTick = performance.now(), previousCpu = cpuUsage(), ticks = 0;
  const timer = setInterval(() => {
    const now = performance.now();
    const lag = Math.max(0, now - lastTick - intervalMs);
    loopLagMs = Math.max(loopLagMs, lag);
    runtimeSample = resourceSample(previousCpu, now - lastTick);
    previousCpu = cpuUsage();
    if (lag >= stallThresholdMs) {
      const stall = {
        observed_at: new Date().toISOString(),
        lag_ms: rounded(lag),
        sample: runtimeSample,
        synchronous_spans: slowSyncSpans.filter((span) =>
          span.end_ms >= lastTick && span.start_ms <= now
        ),
        requests: [...active.values()].slice(0, 16).map(summary),
      };
      loopStalls.push(stall);
      if (loopStalls.length > 8) loopStalls.shift();
      // Emit on recovery, while the just-finished synchronous span is still attributable.
      logger("runtime").warn("event_loop_stall", JSON.stringify(stall));
    }
    lastTick = now;
    if (++ticks % 30 !== 0) return;
    const slow = [...active.values()].filter((t) => now - t.start >= 10000);
    if (slow.length || loopLagMs >= 500) {
      logger("runtime").warn(
        "latency_active",
        JSON.stringify({
          event_loop_max_lag_ms: rounded(loopLagMs),
          runtime_sample: runtimeSample,
          active_count: active.size,
          requests: slow.slice(0, 16).map(summary),
        }),
      );
    }
    previousLoopLagMs = loopLagMs;
    loopLagMs = 0;
  }, intervalMs);
  return () => clearInterval(timer);
}
