import { assertEquals, assertRejects } from "@std/assert";
import { MainDatabase } from "../../src/db/database.ts";
import { MIGRATIONS, REQUIRED_TABLES } from "../../src/db/schema.ts";
import { temporaryDatabase } from "./helpers.ts";

Deno.test("TS-owned database contains exact migrations and required objects", async () => {
  const temporary = await temporaryDatabase();
  try {
    assertEquals(temporary.db.validate().version, 2);
    assertEquals(
      temporary.db.all<{ version: number; checksum: string }>(
        "SELECT version,checksum FROM ts_schema_versions ORDER BY version",
      ),
      MIGRATIONS.map(({ version, checksum }) => ({ version, checksum })),
    );
    for (const table of REQUIRED_TABLES) {
      assertEquals(
        temporary.db.get<{ count: number }>(
          "SELECT COUNT(*) AS count FROM sqlite_master WHERE type='table' AND name=?",
          table,
        )?.count,
        1,
      );
    }
  } finally {
    await temporary.close();
  }
});

Deno.test("db validation refuses checksum drift", async () => {
  const temporary = await temporaryDatabase();
  const path = temporary.path;
  temporary.db.run("UPDATE ts_schema_versions SET checksum='bad' WHERE version=1");
  temporary.db.close();
  const reopened = new MainDatabase(path);
  try {
    await assertRejects(
      () => Promise.resolve().then(() => reopened.validate()),
      Error,
      "checksum mismatch",
    );
  } finally {
    reopened.close();
    await Deno.remove(path.slice(0, path.lastIndexOf("/")), { recursive: true });
  }
});
