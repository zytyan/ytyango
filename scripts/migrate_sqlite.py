#!/usr/bin/env python3
"""Migrate the legacy SQLite schema to the current schema definition.

The script reads data from an existing database that uses the legacy tables
defined in the original project (group_infos, yt_dl_dbs, etc.) and writes data
into a brand new database that follows the SQL definitions under sql/schema_*.sql.
Saved-message tables are skipped by default because they can be extremely large
and are now managed independently. You can enable them explicitly with a flag.
"""
from __future__ import annotations

import argparse
import sqlite3
from pathlib import Path
from typing import Dict, List, Optional, Sequence, Tuple

REPO_ROOT = Path(__file__).resolve().parent.parent
SQL_DIR = REPO_ROOT / "sql"

SCHEMA_FILES = [
    SQL_DIR / "schema_user.sql",
    SQL_DIR / "schema_chat.sql",
    SQL_DIR / "schema_coc.sql",
    SQL_DIR / "schema_bilibili.sql",
    SQL_DIR / "schema_pics.sql",
    SQL_DIR / "schema_ytdl.sql",
]
SAVED_MESSAGE_SCHEMA = SQL_DIR / "schema_saved_msg.sql"


class MigrationError(RuntimeError):
    """Raised when the migration cannot complete."""


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Migrate the old SQLite database to the new schema."
    )
    parser.add_argument(
        "source",
        type=Path,
        help="Path to the legacy SQLite database file",
    )
    parser.add_argument(
        "target",
        type=Path,
        help="Path where the migrated database will be created",
    )
    parser.add_argument(
        "--include-saved-messages",
        action="store_true",
        help="Also create and migrate saved message tables",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Overwrite the target database if it already exists",
    )
    return parser.parse_args()


def _connect(db_path: Path) -> sqlite3.Connection:
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA foreign_keys = ON;")
    return conn


def _ensure_target(path: Path, force: bool) -> None:
    if path.exists():
        if not force:
            raise MigrationError(
                f"Target database {path} already exists. "
                "Use --force to overwrite it."
            )
        path.unlink()
    path.parent.mkdir(parents=True, exist_ok=True)


def exec_schema(conn: sqlite3.Connection, include_saved_msgs: bool) -> None:
    schema_files: List[Path] = list(SCHEMA_FILES)
    if include_saved_msgs:
        schema_files.append(SAVED_MESSAGE_SCHEMA)
    for schema_file in schema_files:
        print(schema_file)
        sql = schema_file.read_text(encoding="utf-8")
        conn.executescript(sql)


def table_exists(conn: sqlite3.Connection, table_name: str) -> bool:
    cur = conn.execute(
        "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?",
        (table_name,),
    )
    return cur.fetchone() is not None


def copy_table(
    src: sqlite3.Connection,
    dst: sqlite3.Connection,
    source_table: str,
    src_columns: Sequence[str],
    dest_table: Optional[str] = None,
    dest_columns: Optional[Sequence[str]] = None,
) -> int:
    if not table_exists(src, source_table):
        return 0
    dest_table = dest_table or source_table
    dest_columns = dest_columns or src_columns
    if len(src_columns) != len(dest_columns):
        raise MigrationError(
            f"Column count mismatch while copying {source_table} -> {dest_table}"
        )
    select_sql = f"SELECT {', '.join(src_columns)} FROM {source_table}"
    rows = src.execute(select_sql).fetchall()
    if not rows:
        return 0
    placeholders = ", ".join("?" for _ in dest_columns)
    insert_sql = (
        f"INSERT INTO {dest_table} ({', '.join(dest_columns)}) "
        f"VALUES ({placeholders})"
    )
    dst.executemany(
        insert_sql,
        ([row[col] for col in src_columns] for row in rows),
    )
    return len(rows)


def to_bool_int(value: Optional[object], default: int = 0) -> int:
    if value is None:
        return default
    if isinstance(value, bool):
        return int(value)
    if isinstance(value, (int, float)):
        return 1 if int(value) != 0 else 0
    if isinstance(value, str):
        normalized = value.strip().lower()
        return 1 if normalized in {"1", "true", "t", "y", "yes", "on"} else 0
    return default


def to_int(value: Optional[object], default: int = 0) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def null_if_zero(value: Optional[object]) -> Optional[int]:
    if value is None:
        return None
    val = to_int(value, default=0)
    return None if val == 0 else val


def migrate_users(src: sqlite3.Connection, dst: sqlite3.Connection) -> int:
    if not table_exists(src, "users"):
        return 0
    rows = src.execute(
        """
        SELECT id, updated_at, user_id, first_name, last_name,
               profile_update_at, profile_photo, time_zone
        FROM users
        """
    ).fetchall()
    if not rows:
        return 0
    insert_sql = """
        INSERT INTO users
        (id, updated_at, user_id, first_name, last_name,
         profile_update_at, profile_photo, timezone)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    """
    payload = []
    for row in rows:
        user_id = to_int(row["user_id"])
        if user_id == 0:
            continue
        db_id = to_int(row["id"])
        timezone = row["time_zone"]
        if timezone is None:
            timezone = 480
        else:
            timezone = to_int(timezone, default=480)
        payload.append(
            (
                db_id,
                to_int(row["updated_at"]),
                user_id,
                row["first_name"] or "",
                row["last_name"],
                to_int(row["profile_update_at"]),
                row["profile_photo"],
                timezone,
            )
        )
    if not payload:
        return 0
    dst.executemany(insert_sql, payload)
    return len(payload)


def migrate_chat_cfg(src: sqlite3.Connection, dst: sqlite3.Connection) -> int:
    if not table_exists(src, "group_infos"):
        return 0
    rows = src.execute(
        """
        SELECT group_id,
               group_web_id,
               auto_cvt_bili,
               auto_ocr,
               auto_calculate,
               auto_exchange,
               auto_check_adult,
               save_messages,
               co_c_enabled,
               resp_nsfw_msg
        FROM group_infos
        """
    ).fetchall()
    if not rows:
        return 0
    insert_sql = """
        INSERT INTO chat_cfg
        (id, web_id, auto_cvt_bili, auto_ocr, auto_calculate, auto_exchange,
         auto_check_adult, save_messages, enable_coc, resp_nsfw_msg)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    """
    payload = []
    for row in rows:
        payload.append(
            (
                row["group_id"],
                null_if_zero(row["group_web_id"]),
                to_bool_int(row["auto_cvt_bili"]),
                to_bool_int(row["auto_ocr"]),
                to_bool_int(row["auto_calculate"]),
                to_bool_int(row["auto_exchange"]),
                to_bool_int(row["auto_check_adult"]),
                to_bool_int(row["save_messages"], default=1),
                to_bool_int(row["co_c_enabled"]),
                to_bool_int(row["resp_nsfw_msg"]),
            )
        )
    dst.executemany(insert_sql, payload)
    return len(rows)


def migrate_character_attrs(
    src: sqlite3.Connection, dst: sqlite3.Connection
) -> int:
    if not table_exists(src, "character_attrs"):
        return 0
    rows = src.execute(
        """
        SELECT user_id, attr_name, attr_value
        FROM character_attrs
        """
    ).fetchall()
    if not rows:
        return 0
    insert_sql = """
        INSERT INTO character_attrs (user_id, attr_name, attr_value)
        VALUES (?, ?, ?)
        ON CONFLICT(user_id, attr_name) DO UPDATE SET attr_value=excluded.attr_value
    """
    payload = []
    for row in rows:
        user_id = row["user_id"]
        if user_id is None:
            continue
        attr_name = row["attr_name"]
        if attr_name is None or attr_name == "":
            continue
        attr_value = row["attr_value"]
        if attr_value is None:
            attr_value = ""
        payload.append((user_id, attr_name, attr_value))
    if not payload:
        return 0
    dst.executemany(insert_sql, payload)
    return len(payload)


def migrate_yt_dl_results(src: sqlite3.Connection, dst: sqlite3.Connection) -> int:
    def read_rows(table: str) -> List[sqlite3.Row]:
        if not table_exists(src, table):
            return []
        return src.execute(
            """
            SELECT url, audio_only, resolution, file_id,
                   title, description, uploader, upload_count
            FROM {table}
            """.format(table=table)
        ).fetchall()

    merged: Dict[Tuple[str, int, int], sqlite3.Row] = {}
    for source_table in ("yt_dl_dbs", "yt_dl_results"):
        for row in read_rows(source_table):
            url = row["url"]
            if not url:
                continue
            key = (
                url,
                to_bool_int(row["audio_only"]),
                to_int(row["resolution"]),
            )
            merged[key] = row

    if not merged:
        return 0

    insert_sql = """
        INSERT INTO yt_dl_results
        (url, audio_only, resolution, file_id, title, description, uploader, upload_count)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(url, audio_only, resolution) DO UPDATE
        SET file_id=excluded.file_id,
            title=excluded.title,
            description=excluded.description,
            uploader=excluded.uploader,
            upload_count=excluded.upload_count
    """
    payload = []
    for row in merged.values():
        payload.append(
            (
                row["url"],
                to_bool_int(row["audio_only"]),
                to_int(row["resolution"]),
                row["file_id"] or "",
                row["title"] or "",
                row["description"] or "",
                row["uploader"] or "",
                to_int(row["upload_count"], default=1),
            )
        )
    dst.executemany(insert_sql, payload)
    return len(payload)


def migrate_saved_msgs(src: sqlite3.Connection, dst: sqlite3.Connection) -> Dict[str, int]:
    stats: Dict[str, int] = {}
    stats["saved_msgs"] = copy_table(
        src,
        dst,
        "saved_msgs",
        (
            "id",
            "mongo_id",
            "peer_id",
            "from_id",
            "msg_id",
            "date",
            "message",
            "image_text",
            "qr_result",
        ),
    )
    stats["saved_msgs_fts"] = copy_table(
        src,
        dst,
        "saved_msgs_fts",
        ("rowid", "doc_id", "message", "image_text", "qr_result"),
        dest_columns=("rowid", "doc_id", "message", "image_text", "qr_result"),
    )
    stats["saved_msgs_fts_state"] = copy_table(
        src,
        dst,
        "saved_msgs_fts_state",
        ("id", "last_saved_msg_id"),
    )
    return stats


def migrate_database(
    source: Path,
    target: Path,
    include_saved_msgs: bool,
) -> Dict[str, int]:
    stats: Dict[str, int] = {}
    with _connect(source) as src_conn, _connect(target) as dst_conn:
        exec_schema(dst_conn, include_saved_msgs)
        stats["users"] = migrate_users(src_conn, dst_conn)
        stats["prpr_caches"] = copy_table(
            src_conn,
            dst_conn,
            "prpr_caches",
            ("profile_photo_uid", "prpr_file_id"),
        )
        stats["chat_cfg"] = migrate_chat_cfg(src_conn, dst_conn)
        stats["bili_inline_results"] = copy_table(
            src_conn,
            dst_conn,
            "bili_inline_results",
            ("uid", "text", "chat_id", "message"),
            dest_columns=("uid", "text", "chat_id", "msg_id"),
        )
        stats["character_attrs"] = migrate_character_attrs(src_conn, dst_conn)
        stats["saved_pics"] = copy_table(
            src_conn,
            dst_conn,
            "saved_pics",
            (
                "file_uid",
                "file_id",
                "bot_rate",
                "rand_key",
                "user_rate",
                "user_rating_sum",
                "rate_user_count",
            ),
        )
        stats["saved_pics_rating"] = copy_table(
            src_conn,
            dst_conn,
            "saved_pics_rating",
            ("file_uid", "user_id", "rating"),
        )
        stats["pic_rate_counter"] = copy_table(
            src_conn,
            dst_conn,
            "pic_rate_counter",
            ("rate", "count"),
        )
        stats["yt_dl_results"] = migrate_yt_dl_results(src_conn, dst_conn)
        if include_saved_msgs:
            stats.update(migrate_saved_msgs(src_conn, dst_conn))
        dst_conn.commit()
    return stats


def summarize(stats: Dict[str, int]) -> None:
    print("Migration finished. Rows copied per table:")
    for table, count in stats.items():
        print(f"  - {table}: {count}")


def main() -> None:
    args = parse_args()
    if not args.source.exists():
        raise MigrationError(f"Source database {args.source} does not exist")
    if args.source.resolve() == args.target.resolve():
        raise MigrationError("Source and target databases must be different files")
    _ensure_target(args.target, force=args.force)
    stats = migrate_database(
        args.source,
        args.target,
        include_saved_msgs=args.include_saved_messages,
    )
    summarize(stats)


if __name__ == "__main__":
    try:
        main()
    except MigrationError as exc:
        print(f"Migration failed: {exc}")
        raise SystemExit(1)
