import { AsyncLocalStorage } from "node:async_hooks";
import { logger } from "../logging.ts";

interface Timing {
  calls: number;
  total_ms: number;
  max_ms: number;
  failures: number;
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

export async function latencyStage<T>(name: string, operation: () => Promise<T>): Promise<T> {
  const trace = scope.getStore();
  if (!trace) return await operation();
  const id = trace.next++, start = performance.now();
  trace.running.set(id, { stage: name, start });
  let failed = false;
  try {
    return await operation();
  } catch (error) {
    failed = true;
    throw error;
  } finally {
    trace.running.delete(id);
    const elapsed = performance.now() - start;
    const previous = trace.stages.get(name) ?? { calls: 0, total_ms: 0, max_ms: 0, failures: 0 };
    previous.calls++;
    previous.total_ms += elapsed;
    previous.max_ms = Math.max(previous.max_ms, elapsed);
    previous.failures += Number(failed);
    trace.stages.set(name, previous);
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
    stages: Object.fromEntries([...trace.stages].map(([name, value]) => [name, {
      ...value,
      total_ms: rounded(value.total_ms),
      max_ms: rounded(value.max_ms),
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
    event_loop_recent_max_lag_ms: rounded(Math.max(loopLagMs, previousLoopLagMs)),
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
    background.set(name, {
      calls: previous.calls + 1,
      last_ms: elapsed,
      max_ms: Math.max(previous.max_ms, elapsed),
    });
  }
}

export function startLatencyMonitor(): () => void {
  let expected = performance.now() + 1000, ticks = 0;
  const timer = setInterval(() => {
    const now = performance.now();
    loopLagMs = Math.max(loopLagMs, now - expected, 0);
    expected = now + 1000;
    if (++ticks % 30 !== 0) return;
    const slow = [...active.values()].filter((t) => now - t.start >= 10000);
    if (slow.length || loopLagMs >= 500) {
      logger("runtime").warn(
        "latency_active",
        JSON.stringify({
          event_loop_max_lag_ms: rounded(loopLagMs),
          active_count: active.size,
          requests: slow.map(summary),
        }),
      );
    }
    previousLoopLagMs = loopLagMs;
    loopLagMs = 0;
  }, 1000);
  return () => clearInterval(timer);
}
