import type { MediaObject, MediaStore } from "../shared/types.ts";

function hex(bytes: Uint8Array): string {
  return Array.from(bytes, (value) => value.toString(16).padStart(2, "0")).join("");
}

async function sha256(bytes: Uint8Array): Promise<string> {
  const stable = Uint8Array.from(bytes);
  return hex(new Uint8Array(await crypto.subtle.digest("SHA-256", stable.buffer)));
}

export class ContentAddressedMediaStore implements MediaStore {
  constructor(readonly root: string) {}

  async put(bytes: Uint8Array, mimeType: string): Promise<MediaObject> {
    const hash = await sha256(bytes);
    const relativePath = `sha256/${hash.slice(0, 2)}/${hash}`;
    const path = `${this.root}/${relativePath}`;
    await Deno.mkdir(path.slice(0, path.lastIndexOf("/")), { recursive: true });
    try {
      const existing = await Deno.readFile(path);
      if (existing.byteLength !== bytes.byteLength || await sha256(existing) !== hash) {
        throw new Error(`media object is corrupt: ${hash}`);
      }
    } catch (error) {
      if (!(error instanceof Deno.errors.NotFound)) throw error;
      const temp = await Deno.makeTempFile({
        dir: path.slice(0, path.lastIndexOf("/")),
        prefix: ".tmp-",
      });
      try {
        const file = await Deno.open(temp, { write: true, truncate: true });
        try {
          let offset = 0;
          while (offset < bytes.length) {
            const written = await file.write(bytes.subarray(offset));
            if (!written) throw new Error("media write made no progress");
            offset += written;
          }
          await file.sync();
        } finally {
          file.close();
        }
        await Deno.rename(temp, path);
      } catch (error) {
        try {
          await Deno.remove(temp);
        } catch {
          // Best-effort cleanup.
        }
        throw error;
      }
    }
    return { sha256: hash, relativePath, byteSize: bytes.byteLength, mimeType };
  }

  async open(hash: string): Promise<Uint8Array> {
    this.#validateHash(hash);
    const bytes = await Deno.readFile(`${this.root}/sha256/${hash.slice(0, 2)}/${hash}`);
    if (await sha256(bytes) !== hash) throw new Error(`media object is corrupt: ${hash}`);
    return bytes;
  }

  async verify(hash: string): Promise<boolean> {
    try {
      await this.open(hash);
      return true;
    } catch {
      return false;
    }
  }

  // Invoke while the runtime is stopped. Catalogued objects are retained even
  // when no current message references them; this does not define history retention.
  async collectGarbage(
    retained: ReadonlySet<string>,
    graceMs: number,
    dryRun = true,
    now = Date.now(),
  ): Promise<Array<{ sha256: string; byteSize: number }>> {
    if (!Number.isFinite(graceMs) || graceMs <= 0) {
      throw new Error("positive media GC grace required");
    }
    const removed: Array<{ sha256: string; byteSize: number }> = [];
    let directories: Deno.DirEntry[];
    try {
      directories = await Array.fromAsync(Deno.readDir(`${this.root}/sha256`));
    } catch (error) {
      if (error instanceof Deno.errors.NotFound) return removed;
      throw error;
    }
    for (const directory of directories) {
      if (!directory.isDirectory || !/^[0-9a-f]{2}$/.test(directory.name)) continue;
      for await (const entry of Deno.readDir(`${this.root}/sha256/${directory.name}`)) {
        if (
          !entry.isFile || !/^[0-9a-f]{64}$/.test(entry.name) ||
          !entry.name.startsWith(directory.name) || retained.has(entry.name)
        ) continue;
        const path = `${this.root}/sha256/${directory.name}/${entry.name}`;
        const stat = await Deno.lstat(path);
        if (!stat.isFile || !stat.mtime || now - stat.mtime.getTime() < graceMs) continue;
        removed.push({ sha256: entry.name, byteSize: stat.size });
        if (!dryRun) await Deno.remove(path);
      }
    }
    return removed;
  }

  #validateHash(hash: string): void {
    if (!/^[0-9a-f]{64}$/.test(hash)) throw new Error("invalid media hash");
  }
}
