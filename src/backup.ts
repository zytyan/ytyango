import { backup } from "node:sqlite";
import { createHash } from "node:crypto";
import { MainDatabase } from "./db/database.ts";
import { SQL } from "./db/queries.ts";
import { DenoProcessRunner } from "./shared/process.ts";
import type { ProcessRunner } from "./shared/types.ts";

// A single SQLite snapshot determines both the database and its media inventory.
export async function createBackup(
  db: MainDatabase,
  mediaRoot: string,
  includeMedia: boolean,
  signal: AbortSignal,
  process: ProcessRunner = new DenoProcessRunner(),
): Promise<{ path: string; cleanup: () => Promise<void> }> {
  const root = await Deno.makeTempDir({ prefix: "ytyan-backup-" });
  const cleanup = () => Deno.remove(root, { recursive: true });
  try {
    signal.throwIfAborted();
    await backup(db.raw, `${root}/main.db`, {
      rate: 128,
      progress: () => signal.throwIfAborted(),
    });
    const snapshot = new MainDatabase(`${root}/main.db`, { readOnly: true });
    const inventory = [];
    try {
      snapshot.validate();
      if (includeMedia) {
        await Deno.mkdir(`${root}/ai-media`);
        for (
          const row of snapshot.all<
            { sha256: string; relative_path: string; byte_size: number; mime_type: string }
          >(SQL.BackupMedia)
        ) {
          signal.throwIfAborted();
          const expected = `sha256/${row.sha256.slice(0, 2)}/${row.sha256}`;
          if (!/^[0-9a-f]{64}$/.test(row.sha256) || row.relative_path !== expected) {
            throw new Error("invalid backup media path");
          }
          const source = `${mediaRoot}/${expected}`;
          const stat = await Deno.lstat(source);
          if (!stat.isFile || stat.size !== row.byte_size) {
            throw new Error("backup media size mismatch");
          }
          const target = `${root}/ai-media/${expected}`;
          await Deno.mkdir(target.slice(0, target.lastIndexOf("/")), { recursive: true });
          const hash = createHash("sha256");
          const input = await Deno.open(source, { read: true });
          const output = await Deno.open(target, { createNew: true, write: true, mode: 0o600 });
          let size = 0;
          try {
            for await (const chunk of input.readable) {
              signal.throwIfAborted();
              hash.update(chunk);
              size += chunk.length;
              let position = 0;
              while (position < chunk.length) {
                position += await output.write(chunk.subarray(position));
              }
            }
          } finally {
            try {
              input.close();
            } catch { /* readable closes itself */ }
            output.close();
          }
          if (size !== row.byte_size || hash.digest("hex") !== row.sha256) {
            throw new Error("backup media hash mismatch");
          }
          inventory.push(row);
        }
      }
    } finally {
      snapshot.close();
    }
    const list = inventory.map((row) =>
      `${row.sha256}\t${row.byte_size}\t${row.mime_type}\t${row.relative_path}\n`
    ).join("");
    if (includeMedia) await Deno.writeTextFile(`${root}/media-manifest.tsv`, list);
    await Deno.writeTextFile(
      `${root}/manifest.json`,
      JSON.stringify({
        schema: "ytyan-ts-backup-v1",
        timestamp: new Date().toISOString(),
        databases: [{
          name: "main",
          path: "main.db",
          size: (await Deno.stat(`${root}/main.db`)).size,
        }],
        complete_ai_dataset: includeMedia,
        media: {
          included: includeMedia,
          object_count: inventory.length,
          total_bytes: inventory.reduce((sum, row) => sum + row.byte_size, 0),
          list_sha256: includeMedia ? createHash("sha256").update(list).digest("hex") : undefined,
        },
      }),
    );
    const path = `${root}/backup.tar.gz`;
    const result = await process.run("/usr/bin/tar", [
      "-czf",
      path,
      "-C",
      root,
      "main.db",
      "manifest.json",
      ...(includeMedia ? ["media-manifest.tsv", "ai-media"] : []),
    ], {
      timeoutMs: 30 * 60_000,
      signal,
    });
    signal.throwIfAborted();
    if (result.code !== 0) throw new Error("backup archive failed");
    return { path, cleanup };
  } catch (error) {
    await cleanup();
    throw error;
  }
}
