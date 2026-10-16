import { assertEquals, assertRejects } from "@std/assert";
import {
  latencySnapshot,
  latencyStage,
  latencySyncStage,
  startLatencyMonitor,
  traceMeasurements,
  traceTags,
  traceUpdate,
  updatesReceived,
} from "../../src/shared/latency.ts";
import { installLogging } from "../../src/logging.ts";
import { KeyedMutex } from "../../src/shared/concurrency.ts";

Deno.test("latency correlates overlapping updates, measures session waiting, and preserves errors", async () => {
  const logging = installLogging(8);
  let release!: () => void;
  const gate = new Promise<void>((resolve) => release = resolve);
  const mutex = new KeyedMutex();
  let ready!: () => void;
  const entered = new Promise<void>((resolve) => ready = resolve);
  const error = new Error("failure text must not enter timing records");
  try {
    updatesReceived([900001, 900002]);
    const first = traceUpdate(900001, "ai", Math.floor(Date.now() / 1000), async () => {
      traceTags({ session_id: 7, provider: "gemini" });
      await mutex.use("7", async () => {
        ready();
        await latencyStage("provider.headers", () => gate);
      }, "queue.ai_session");
    });
    await entered;
    const second = traceUpdate(900002, "ai", undefined, () =>
      mutex.use("7", async () => {
        traceTags({ session_id: 7, provider: "deepseek" });
        await latencyStage("provider.deepseek", () => Promise.reject(error));
      }, "queue.ai_session"));
    await Promise.resolve();
    const pending = latencySnapshot().active;
    assertEquals(
      pending.find((t) => t.update_id === 900001)!.waiting.some((t) =>
        t.stage === "provider.headers"
      ),
      true,
    );
    assertEquals(
      pending.find((t) => t.update_id === 900002)!.waiting.some((t) =>
        t.stage === "queue.ai_session"
      ),
      true,
    );
    release();
    await first;
    await assertRejects(() => second, Error, error.message);
    const records = latencySnapshot().recent;
    const a = records.find((t) => t.update_id === 900001)!;
    const b = records.find((t) => t.update_id === 900002)!;
    assertEquals(a.provider, "gemini");
    assertEquals(b.provider, "deepseek");
    assertEquals(a.runner_queue_ms !== undefined, true);
    assertEquals(a.transport_lag_ms !== undefined, true);
    assertEquals(b.transport_lag_ms, undefined);
    assertEquals(b.stages["provider.deepseek"]!.failures, 1);
    assertEquals(JSON.stringify(records).includes(error.message), false);
    const rejected = { ok: false, error_code: 400 };
    const returned = await traceUpdate(900003, "callback", undefined, () =>
      latencyStage(
        "telegram.answerCallbackQuery",
        () => Promise.resolve(rejected),
        (value) => !value.ok,
      ));
    assertEquals(returned, rejected);
    assertEquals(
      latencySnapshot().recent.find((t) => t.update_id === 900003)!
        .stages["telegram.answerCallbackQuery"]!.failures,
      1,
    );
    assertEquals(
      latencySnapshot().active.some((t) => [900001, 900002].includes(t.update_id)),
      false,
    );
  } finally {
    release();
    logging.restore();
  }
});

Deno.test("latency history is bounded and untraced stages do not create records", async () => {
  const logging = installLogging(8);
  try {
    const before = latencySnapshot().recent.length;
    await latencyStage("untracked", () => Promise.resolve());
    assertEquals(latencySnapshot().recent.length, before);
    for (let id = 0; id < 105; id++) {
      await traceUpdate(910000 + id, "command", undefined, () => Promise.resolve());
    }
    assertEquals(latencySnapshot().recent.length, 100);
    assertEquals(latencySnapshot().recent.some((t) => t.update_id === 910000), false);
  } finally {
    logging.restore();
  }
});

Deno.test("latency records stage start offsets and synchronous database work without retaining values", async () => {
  const logging = installLogging(8);
  try {
    const secret = "query-value-not-for-diagnostics";
    await traceUpdate(920001, "ai", undefined, async () => {
      const value = latencySyncStage("db.get.RunForRequest", () => secret);
      assertEquals(value, secret);
      await latencyStage("ai.reaction", () => Promise.resolve());
      try {
        latencySyncStage("db.run.FreezeRunContext", () => {
          throw new Error(secret);
        });
      } catch { /* Preserve the failure marker, without logging the error body. */ }
    });
    const trace = latencySnapshot().recent.find((t) => t.update_id === 920001)!;
    const db = trace.stages["db.get.RunForRequest"]!;
    const reaction = trace.stages["ai.reaction"]!;
    assertEquals(db.first_start_ms >= 0, true);
    assertEquals(db.last_end_ms >= db.first_start_ms, true);
    assertEquals(reaction.first_start_ms >= db.last_end_ms, true);
    assertEquals(reaction.last_end_ms >= reaction.first_start_ms, true);
    assertEquals(trace.stages["db.run.FreezeRunContext"]!.failures, 1);
    assertEquals(JSON.stringify(trace).includes(secret), false);
  } finally {
    logging.restore();
  }
});

Deno.test("loop stall probe attributes a completed synchronous span and samples CPU/memory under unit permissions", async () => {
  const logging = installLogging(8);
  const stop = startLatencyMonitor(10, 5);
  try {
    await traceUpdate(930001, "ai", undefined, () => {
      traceTags({ run_id: 123 });
      traceMeasurements({ history_messages: 3, context_snapshot_chars: 100 });
      latencySyncStage("probe.synthetic", () => {
        const deadline = performance.now() + 70;
        while (performance.now() < deadline) { /* Deliberately block to exercise the probe. */ }
      });
      return Promise.resolve();
    });
    await new Promise((resolve) => setTimeout(resolve, 25));
    const snapshot = latencySnapshot();
    const span = snapshot.slow_synchronous_spans.find((s) => s.update_id === 930001)!;
    assertEquals(span.stage, "probe.synthetic");
    assertEquals(span.run_id, 123);
    assertEquals(span.elapsed_ms >= 50, true);
    const stall = snapshot.event_loop_stalls.find((s) =>
      s.synchronous_spans.some((p) => p.update_id === 930001)
    )!;
    assertEquals(stall.lag_ms >= 5, true);
    assertEquals(stall.sample.cpu_user_ms >= 0, true);
    assertEquals(stall.sample.heap_used_bytes > 0, true);
    assertEquals(stall.requests.some((r) => r.update_id === 930001), false);
    assertEquals(snapshot.recent.find((r) => r.update_id === 930001)!.measurements, {
      history_messages: 3,
      context_snapshot_chars: 100,
    });
    assertEquals(snapshot.slow_synchronous_spans.length <= 64, true);
    assertEquals(snapshot.event_loop_stalls.length <= 8, true);
  } finally {
    stop();
    logging.restore();
  }
});
