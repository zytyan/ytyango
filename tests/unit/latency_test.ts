import { assertEquals, assertRejects } from "@std/assert";
import {
  latencySnapshot,
  latencyStage,
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
