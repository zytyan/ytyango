import { assertEquals, assertRejects, assertThrows } from "@std/assert";
import { innerAddress, innerHandler } from "../../src/inner_http.ts";
import { createBackup } from "../../src/backup.ts";
import { StatsService } from "../../src/stats/service.ts";
import { ContentAddressedMediaStore } from "../../src/ai/media_store.ts";
import { temporaryDatabase } from "./helpers.ts";

Deno.test("inner HTTP authenticates every route and validates counters and log controls", async () => {
  assertEquals(innerAddress(), { hostname: "127.0.0.1", port: 4019 });
  assertEquals(innerAddress("OFF"), undefined);
  assertThrows(() => innerAddress("0.0.0.0:4019"));
  const temporary = await temporaryDatabase();
  const stats = new StatsService(temporary.db);
  let level = 0;
  const handler = innerHandler({
    db: temporary.db,
    stats,
    mediaRoot: "unused",
    token: "secret",
    signal: new AbortController().signal,
    logging: {
      get: () => level,
      set: (value) => {
        level = value;
      },
    },
  });
  const request = (path: string, method = "GET", value?: unknown) =>
    new Request(`http://localhost${path}`, {
      method,
      headers: { "X-Backup-Token": "secret" },
      body: value === undefined ? undefined : JSON.stringify(value),
    });
  try {
    for (const path of ["/", "/loggers", "/backupdb", "/debug/profile", "/mars-counter"]) {
      assertEquals((await handler(new Request(`http://localhost${path}`))).status, 401);
    }
    assertEquals(
      (await handler(request("/mars-counter", "POST", { group_id: -100, mars_count: 9 }))).status,
      200,
    );
    assertEquals(
      (await handler(request("/dio-ban", "POST", { group_id: -100, user_id: 5, action: 2 })))
        .status,
      200,
    );
    assertEquals(
      (await handler(request("/dio-ban", "POST", { group_id: -100, user_id: 5, action: "wrong" })))
        .status,
      400,
    );
    assertEquals((await handler(request("/loggers/runtime/4", "PUT"))).status, 204);
    assertEquals(level, 4);
    assertEquals((await handler(request("/backupdb?db=msg"))).status, 410);
    assertEquals((await handler(request("/backupdb?media=bad"))).status, 400);
    assertEquals((await handler(request("/debug/profile?seconds=0"))).status, 400);
    stats.flush();
    assertEquals(
      temporary.db.get(
        "SELECT mars_count,max_mars_count,dio_ban_user_count FROM chat_stat_daily WHERE chat_id=-100",
      ),
      { mars_count: 1, max_mars_count: 9, dio_ban_user_count: 1 },
    );
  } finally {
    await temporary.close();
  }
});

Deno.test("backup uses an independent consistent snapshot and verifies media before archive", async () => {
  const temporary = await temporaryDatabase();
  const mediaRoot = await Deno.makeTempDir();
  const store = new ContentAddressedMediaStore(mediaRoot);
  const object = await store.put(new Uint8Array([1, 2, 3]), "image/jpeg");
  temporary.db.run(
    "INSERT INTO media_objects VALUES(?,?,?,?,?)",
    object.sha256,
    object.relativePath,
    object.byteSize,
    "image/jpeg",
    1,
  );
  let calls = 0;
  const process = {
    async run(_command: string, args: readonly string[]) {
      calls++;
      const root = args[args.indexOf("-C") + 1]!;
      const manifest = JSON.parse(await Deno.readTextFile(`${root}/manifest.json`));
      assertEquals(manifest.complete_ai_dataset, true);
      assertEquals(manifest.media.object_count, 1);
      assertEquals(
        await Deno.readFile(`${root}/ai-media/${object.relativePath}`),
        new Uint8Array([1, 2, 3]),
      );
      // A later write never appears in the snapshot chosen for this backup.
      temporary.db.run("INSERT INTO users VALUES(77,1,'late',NULL,NULL)");
      const { MainDatabase } = await import("../../src/db/database.ts");
      const snapshot = new MainDatabase(`${root}/main.db`, { readOnly: true });
      try {
        assertEquals(snapshot.get("SELECT * FROM users WHERE user_id=77"), undefined);
      } finally {
        snapshot.close();
      }
      await Deno.writeFile(args[1]!, new Uint8Array([0]));
      return { code: 0, stdout: new Uint8Array(), stderr: new Uint8Array() };
    },
  };
  try {
    const result = await createBackup(
      temporary.db,
      mediaRoot,
      true,
      new AbortController().signal,
      process,
    );
    assertEquals(calls, 1);
    await result.cleanup();
    await Deno.writeFile(`${mediaRoot}/${object.relativePath}`, new Uint8Array([4, 5, 6]));
    await assertRejects(() =>
      createBackup(temporary.db, mediaRoot, true, new AbortController().signal, process)
    );
    assertEquals(calls, 1);
    await assertRejects(() =>
      createBackup(temporary.db, mediaRoot, false, AbortSignal.abort(), process)
    );
  } finally {
    await temporary.close();
    await Deno.remove(mediaRoot, { recursive: true });
  }
});

Deno.test("media GC keeps catalogue objects and young files and defaults to dry run", async () => {
  const root = await Deno.makeTempDir();
  try {
    const store = new ContentAddressedMediaStore(root);
    const retained = await store.put(new Uint8Array([1]), "image/jpeg");
    const orphan = await store.put(new Uint8Array([2]), "image/jpeg");
    await store.put(new Uint8Array([3]), "image/jpeg");
    const old = new Date(Date.now() - 7200_000);
    for (const object of [retained, orphan]) {
      await Deno.utime(`${root}/${object.relativePath}`, old, old);
    }
    const keep = new Set([retained.sha256]);
    assertEquals((await store.collectGarbage(keep, 3600_000)).map((row) => row.sha256), [
      orphan.sha256,
    ]);
    assertEquals(await store.verify(orphan.sha256), true);
    await store.collectGarbage(keep, 3600_000, false);
    assertEquals(await store.verify(orphan.sha256), false);
    assertEquals(await store.verify(retained.sha256), true);
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("media publication handles partial filesystem writes before rename", async () => {
  const root = await Deno.makeTempDir();
  const originalOpen = Deno.open;
  try {
    Deno.open = async (path, options) => {
      const file = await originalOpen(path, options);
      if (options?.write) {
        const write = file.write.bind(file);
        file.write = (bytes) => write(bytes.subarray(0, 2));
      }
      return file;
    };
    const store = new ContentAddressedMediaStore(root), bytes = new Uint8Array([1, 2, 3, 4, 5]);
    const object = await store.put(bytes, "image/jpeg");
    assertEquals(await store.open(object.sha256), bytes);
  } finally {
    Deno.open = originalOpen;
    await Deno.remove(root, { recursive: true });
  }
});
