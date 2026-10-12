import { resolve } from "node:path";
import { createHash } from "node:crypto";
import { SQL } from "./queries.ts";
import { MainDatabase } from "./database.ts";
import { ContentAddressedMediaStore } from "../ai/media_store.ts";

type Value = null | number | bigint | string | Uint8Array;
type Row = Record<string, Value>;
const quote = (value: string) => `"${value.replaceAll('"', '""')}"`;
const encode = (value: unknown): string =>
  JSON.stringify(
    value,
    (_key, item) =>
      item instanceof Uint8Array
        ? { blobHex: Array.from(item, (b) => b.toString(16).padStart(2, "0")).join("") }
        : typeof item === "bigint"
        ? { integer: String(item) }
        : item && typeof item === "object" && !Array.isArray(item)
        ? Object.fromEntries(Object.entries(item).sort(([a], [b]) => a.localeCompare(b)))
        : item,
  );
function rows64(db: MainDatabase, table: string): Iterable<Row> {
  const columns = db.all<{ name: string; pk: number }>(`PRAGMA table_xinfo(${quote(table)})`);
  const keys = columns.filter((column) => column.pk > 0).sort((a, b) => a.pk - b.pk).map((column) =>
    column.name
  );
  const order = keys.length ? keys : columns.map((column) => column.name);
  const statement = db.raw.prepare(
    `SELECT * FROM ${quote(table)} ORDER BY ${order.map(quote).join(",")}`,
  );
  statement.setReadBigInts(true);
  return statement.iterate() as Iterable<Row>;
}
function digest(rows: Iterable<Row>): { sha256: string; count: number } {
  const digest = createHash("sha256");
  let count = 0;
  for (const row of rows) {
    count++;
    for (const key of Object.keys(row).sort()) {
      const value = row[key];
      const bytes = value instanceof Uint8Array ? value : new TextEncoder().encode(encode(value));
      digest.update(
        `${key}:${value instanceof Uint8Array ? "blob" : "value"}:${bytes.byteLength}:`,
      );
      digest.update(bytes);
    }
    digest.update("\n");
  }
  return { sha256: digest.digest("hex"), count };
}
function archivedRows(db: MainDatabase, table: string): Iterable<Row> {
  const statement = db.raw.prepare(
    "SELECT payload FROM import_archives WHERE source_table=? ORDER BY CAST(row_key AS INTEGER)",
  );
  return Array.from(
    statement.iterate(table),
    (row) =>
      JSON.parse(String(row.payload), (_key, value) => {
        if (value && typeof value === "object" && "integer" in value) return BigInt(value.integer);
        if (value && typeof value === "object" && "blobHex" in value) {
          return Uint8Array.from(value.blobHex.match(/.{2}/g) ?? [], (hex: string) =>
            parseInt(hex, 16));
        }
        return value;
      }) as Row,
  );
}

export interface ImportOptions {
  source: string;
  output: string;
  media?: string;
}

/** All writes are confined to a newly reserved output directory. */
export async function importDatabase(options: ImportOptions): Promise<Record<string, unknown>> {
  const output = resolve(options.output);
  const sourcePath = resolve(options.source);
  const source = new MainDatabase(sourcePath, { readOnly: true });
  let reserved = false;
  let target: MainDatabase | undefined;
  try {
    source.raw.exec("PRAGMA query_only=ON; BEGIN");
    const tables = source.all<{ name: string }>(
      "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name",
    ).map((row) => row.name);
    if (!tables.includes("schema_migrations")) {
      throw new Error(
        "source must be Go/legacy TS canonical V9–V11; migrate older sources with Go offline tools first",
      );
    }
    const migrations = source.all<Row>("SELECT * FROM schema_migrations ORDER BY version");
    const version = Number(migrations.at(-1)?.version);
    if (![9, 10, 11].includes(version) || migrations.length !== version) {
      throw new Error("unsupported or incomplete source migration history");
    }
    for (
      const name of [
        "ai_sessions",
        "ai_messages",
        "ai_runs",
        "ai_session_messages",
        "media_objects",
        "ai_message_media",
        "chat_cfg",
        "chat_stat_daily",
        "saved_pics",
        "saved_pics_rating",
        "pic_rate_counter",
      ]
    ) {
      if (!tables.includes(name)) throw new Error(`source is missing retained table: ${name}`);
    }
    if (
      source.get<{ integrity_check: string }>("PRAGMA integrity_check")?.integrity_check !== "ok" ||
      source.all("PRAGMA foreign_key_check").length
    ) throw new Error("source integrity/foreign keys failed");
    // mkdir is exclusive: neither existing targets nor media trees can be overwritten.
    await Deno.mkdir(output);
    reserved = true;
    const stage = `${output}/.staging`;
    await Deno.mkdir(stage);
    target = new MainDatabase(`${stage}/main.db`);
    target.initialize();
    const targetTables = new Set(
      target.all<{ name: string }>("SELECT name FROM sqlite_master WHERE type='table'").map((row) =>
        row.name
      ),
    );
    const report: Array<Record<string, unknown>> = [];
    target.transaction(() => {
      target!.raw.exec("PRAGMA defer_foreign_keys=ON");
      // Bulk import copies existing rating aggregates; triggers resume after loading votes.
      const triggers = target!.all<{ name: string; sql: string }>(
        "SELECT name,sql FROM sqlite_master WHERE type='trigger'",
      );
      for (const trigger of triggers) target!.raw.exec(`DROP TRIGGER ${quote(trigger.name)}`);
      for (const name of tables) {
        if (name === "schema_migrations") continue;
        const rows = rows64(source, name);
        const destination = name === "character_attrs" ? "legacy_coc_attrs" : name;
        if (!targetTables.has(destination) || name === "ai_session_provider_state") {
          // Source-only data, including albums, is explicitly archived rather than silently discarded.
          const insert = target!.raw.prepare(
            "INSERT INTO import_archives(source_table,row_key,payload) VALUES(?,?,?)",
          );
          let count = 0;
          for (const row of rows) insert.run(name, String(count++), encode(row));
          report.push({
            source: name,
            destination: "import_archives",
            count,
            action: "archive",
          });
          continue;
        }
        const info = target!.all<{ name: string; hidden: number }>(
          `PRAGMA table_xinfo(${quote(destination)})`,
        );
        const columns = info.filter((column) => column.hidden === 0).map((column) => column.name);
        const sourceColumns = source.all<{ name: string }>(`PRAGMA table_xinfo(${quote(name)})`)
          .map((column) => column.name);
        if (columns.some((column) => !sourceColumns.includes(column))) {
          throw new Error(`missing columns in ${name}`);
        }
        const insert = target!.raw.prepare(
          `INSERT INTO ${quote(destination)}(${columns.map(quote).join(",")}) VALUES(${
            columns.map(() => "?").join(",")
          })`,
        );
        let count = 0;
        for (const row of rows) {
          insert.run(...columns.map((column) => row[column]!));
          count++;
        }
        report.push({
          source: name,
          destination,
          count,
          action: name === "character_attrs" ? "archive" : "identity",
        });
      }
      for (const trigger of triggers) target!.raw.exec(trigger.sql);
    });
    for (const mapping of report) {
      const name = String(mapping.source);
      const rows = rows64(source, name);
      const copied = mapping.destination === "import_archives"
        ? archivedRows(target, name)
        : rows64(target, String(mapping.destination));
      const original = digest(rows);
      const replacement = digest(copied);
      if (original.sha256 !== replacement.sha256 || original.count !== replacement.count) {
        throw new Error(`row/payload validation failed for ${name}`);
      }
      mapping.sha256 = original.sha256;
    }
    const tokens = source.get<Row>(
      "SELECT COALESCE(SUM(total_input_tokens),0) AS input,COALESCE(SUM(total_output_tokens),0) AS output,COALESCE(SUM(total_cached_input_tokens),0) AS cached FROM ai_sessions",
    );
    const albumMapping = {
      groups: 0,
      files: 0,
      topic: "unknown legacy topic (-1); selected only by Telegram group/message identity",
      captions: "preserved in import_archives only",
    };
    if (tables.includes("ai_media_groups") && tables.includes("ai_media_group_photos")) {
      target.transaction(() => {
        for (const row of rows64(source, "ai_media_groups")) {
          target!.run(
            SQL.IndexAlbum,
            row.chat_id!,
            -1,
            row.media_group_id!,
            row.expires_at!,
            row.photo_only!,
          );
          albumMapping.groups++;
        }
        for (const row of rows64(source, "ai_media_group_photos")) {
          target!.run(
            SQL.IndexAlbumFile,
            row.chat_id!,
            -1,
            row.media_group_id!,
            row.msg_id!,
            row.telegram_file_id!,
            "",
            null,
          );
          albumMapping.files++;
        }
      });
      for (
        const [table, sql] of [["ai_media_groups", SQL.ImportedAlbumGroups], [
          "ai_media_group_photos",
          SQL.ImportedAlbumFiles,
        ]]
      ) {
        const mapped = function* (): Iterable<Row> {
          for (const row of rows64(source, table!)) {
            yield table === "ai_media_groups"
              ? {
                chat_id: row.chat_id!,
                topic_id: -1n,
                group_id: row.media_group_id!,
                expires_at: row.expires_at!,
                photo_only: row.photo_only!,
              }
              : {
                chat_id: row.chat_id!,
                topic_id: -1n,
                group_id: row.media_group_id!,
                msg_id: row.msg_id!,
                file_id: row.telegram_file_id!,
                file_unique_id: "",
                byte_size: null,
              };
          }
        };
        const statement = target.raw.prepare(sql!);
        statement.setReadBigInts(true);
        const expected = digest(mapped()), actual = digest(statement.iterate() as Iterable<Row>);
        if (expected.count !== actual.count || expected.sha256 !== actual.sha256) {
          throw new Error("album index mapping validation failed");
        }
      }
    }
    const media = target.all<
      { sha256: string; byte_size: number; relative_path: string; mime_type: string }
    >("SELECT sha256,byte_size,relative_path,mime_type FROM media_objects ORDER BY sha256");
    if (media.length && !options.media) {
      throw new Error("source references media; --media source directory is required");
    }
    const oldMedia = new ContentAddressedMediaStore(options.media ?? "");
    const newMedia = new ContentAddressedMediaStore(`${stage}/ai-media`);
    await Deno.mkdir(newMedia.root, { recursive: true });
    for (const item of media) {
      if (item.relative_path !== `sha256/${item.sha256.slice(0, 2)}/${item.sha256}`) {
        throw new Error("noncanonical source media path");
      }
      const bytes = await oldMedia.open(item.sha256);
      if (bytes.length !== item.byte_size) throw new Error(`media size mismatch: ${item.sha256}`);
      const copied = await newMedia.put(bytes, item.mime_type);
      if (copied.sha256 !== item.sha256 || !await newMedia.verify(item.sha256)) {
        throw new Error("media copy validation failed");
      }
    }
    const checked = target.validate();
    const manifest = {
      format: "ytyanbot-ts-import-v1",
      source: sourcePath,
      sourceVersion: version,
      sourceMigrations: migrations,
      destination: `${output}/dataset/main.db`,
      idMapping: "retained chat/message/session/run IDs",
      tables: report,
      albumMapping,
      tokens,
      media: { count: media.length, bytes: media.reduce((sum, item) => sum + item.byte_size, 0) },
      validation: checked,
      providerState: "archived; new requests rebuild from preserved native history",
      createdAt: new Date().toISOString(),
    };
    target.raw.exec("PRAGMA wal_checkpoint(TRUNCATE)");
    target.close();
    target = undefined;
    await Deno.writeTextFile(`${stage}/migration-report.json`, JSON.stringify(manifest, null, 2));
    await Deno.rename(stage, `${output}/dataset`);
    return manifest;
  } catch (error) {
    target?.close();
    target = undefined;
    if (reserved) await Deno.remove(output, { recursive: true });
    throw error;
  } finally {
    source.close();
  }
}
