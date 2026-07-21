import type { ProcessResult, ProcessRunner } from "./types.ts";

export class DenoProcessRunner implements ProcessRunner {
  async run(command: string, args: readonly string[], options: {
    cwd?: string;
    timeoutMs: number;
    signal?: AbortSignal;
  }): Promise<ProcessResult> {
    if (options.signal?.aborted) throw options.signal.reason;
    const child = new Deno.Command(command, {
      args: [...args],
      cwd: options.cwd,
      stdin: "null",
      stdout: "piped",
      stderr: "piped",
    }).spawn();
    const timeout = setTimeout(() => {
      try {
        child.kill("SIGKILL");
      } catch {
        // The process may have exited between the timeout and kill.
      }
    }, options.timeoutMs);
    const abort = () => {
      try {
        child.kill("SIGTERM");
      } catch {
        // The process is already gone.
      }
    };
    options.signal?.addEventListener("abort", abort, { once: true });
    try {
      const result = await child.output();
      if (options.signal?.aborted) throw options.signal.reason;
      return { code: result.code, stdout: result.stdout, stderr: result.stderr };
    } finally {
      clearTimeout(timeout);
      options.signal?.removeEventListener("abort", abort);
    }
  }
}
