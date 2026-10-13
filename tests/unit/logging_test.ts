import { assertEquals } from "@std/assert";
import { installLogging, logger } from "../../src/logging.ts";

Deno.test("module logging can override the runtime threshold and follows config changes independently", () => {
  const original = {
    debug: console.debug,
    log: console.log,
    info: console.info,
    warn: console.warn,
    error: console.error,
  };
  const calls: unknown[][] = [];
  for (const name of Object.keys(original) as (keyof typeof original)[]) {
    console[name] = (...args) => {
      calls.push(args);
    };
  }
  let configured = 8;
  const control = installLogging(() => configured);
  try {
    const ai = logger("ai"), download = logger("download");
    control.set(-4, "ai");
    ai.debug("trace");
    download.warn("hidden");
    assertEquals(calls, [["[ai]", "trace"]]);
    configured = 4;
    download.warn("visible");
    assertEquals(control.get("download"), 4);
    control.set(8, "runtime");
    ai.debug("still visible");
    download.warn("hidden again");
    assertEquals(calls, [["[ai]", "trace"], ["[download]", "visible"], ["[ai]", "still visible"]]);
  } finally {
    control.restore();
    Object.assign(console, original);
  }
});
