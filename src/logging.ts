export const LOG_NAMES = [
  "runtime",
  "ai",
  "download",
  "moderation",
  "stats",
  "inner-http",
] as const;
export type LogName = typeof LOG_NAMES[number];
const severity = { debug: -4, log: 0, info: 0, warn: 4, error: 8 } as const;
type Method = keyof typeof severity;

export function logger(name: LogName) {
  const write = (method: Method) => (...args: unknown[]) => console[method](`[${name}]`, ...args);
  return { debug: write("debug"), info: write("info"), warn: write("warn"), error: write("error") };
}

export function installLogging(initial: number | (() => number)) {
  const overrides = new Map<LogName, number>();
  const get = (name: LogName = "runtime") =>
    overrides.get(name) ?? overrides.get("runtime") ??
      (typeof initial === "function" ? initial() : initial);
  const original = {
    debug: console.debug,
    log: console.log,
    info: console.info,
    warn: console.warn,
    error: console.error,
  };
  for (const method of Object.keys(severity) as Method[]) {
    console[method] = (...args: unknown[]) => {
      const name = LOG_NAMES.find((name) => args[0] === `[${name}]`) ?? "runtime";
      if (severity[method] >= get(name)) original[method](...args);
    };
  }
  return {
    get,
    set(value: number, name: LogName = "runtime") {
      overrides.set(name, value);
    },
    restore: () => Object.assign(console, original),
  };
}
export type LoggingControl = Pick<ReturnType<typeof installLogging>, "get" | "set">;
