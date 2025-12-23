#!/usr/bin/env python3
"""
Migrate data from the legacy SQLite database into PostgreSQL.

Usage:
  PGURL=postgres://user:pass@host:5433/dbname?sslmode=disable python migrate_sqlite_to_pg.py ytyan_new_backup.db --include-saved-msgs

Notes:
- Passwords are read from the supplied PG URL or environment; do not hardcode them.
- The target PostgreSQL instance must be reachable; tables will be created if they do not exist.
"""

from __future__ import annotations

import argparse
import sqlite3
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterable, List, Sequence, Tuple

import psycopg2
from psycopg2 import errorcodes, extras

REPO_ROOT = Path(__file__).resolve().parent.parent
SQL_DIR = REPO_ROOT / "sql"

SCHEMA_FILES_BASE = [
    SQL_DIR / "schema_user.sql",
    SQL_DIR / "schema_chat.sql",
    SQL_DIR / "schema_coc.sql",
    SQL_DIR / "schema_pics.sql",
    SQL_DIR / "schema_bilibili.sql",
    SQL_DIR / "schema_ytdl.sql",
    SQL_DIR / "schema_gemini.sql",
]
SAVED_MSG_SCHEMA = SQL_DIR / "schema_saved_msg.sql"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Migrate SQLite data into PostgreSQL.")
    parser.add_argument("sqlite_path", type=Path, help="Path to source SQLite database (e.g., ytyan_new_backup.db)")
    parser.add_argument(
        "--pg-url",
        dest="pg_url",
        help="PostgreSQL connection URL (or set PGURL/DATABASE_URL env).",
        default=None,
    )
    parser.add_argument(
        "--include-saved-msgs",
        action="store_true",
        help="Also migrate saved_msgs/raw_update/edit_history tables.",
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=500,
        help="Batch size for inserts (default: 500).",
    )
    return parser.parse_args()


def load_pg_url(args: argparse.Namespace) -> str:
    import os

    url = args.pg_url or os.getenv("PGURL") or os.getenv("DATABASE_URL")
    if not url:
        raise RuntimeError("PostgreSQL URL not provided. Set --pg-url or PGURL/DATABASE_URL env.")
    return url


def connect_sqlite(path: Path) -> sqlite3.Connection:
    conn = sqlite3.connect(path)
    conn.row_factory = sqlite3.Row
    return conn


def connect_pg(url: str):
    return psycopg2.connect(url)


def table_exists(sqlite_conn: sqlite3.Connection, name: str) -> bool:
    cur = sqlite_conn.execute(
        "SELECT 1 FROM sqlite_master WHERE type='table' AND name = ? LIMIT 1", (name,),
    )
    return cur.fetchone() is not None


def exec_schema(pg_conn, include_saved_msgs: bool) -> None:
    files: List[Path] = list(SCHEMA_FILES_BASE)
    if include_saved_msgs:
        files.append(SAVED_MSG_SCHEMA)
    duplicate_sqlstates = {
        errorcodes.DUPLICATE_TABLE,
        errorcodes.DUPLICATE_FUNCTION,
        errorcodes.DUPLICATE_OBJECT,  # trigger/constraint/index
    }
    with pg_conn:
        for f in files:
            sql = f.read_text(encoding="utf-8")
            with pg_conn.cursor() as cur:
                try:
                    cur.execute(sql)
                except psycopg2.Error as e:
                    code = getattr(e, "pgcode", None)
                    if code in duplicate_sqlstates:
                        pg_conn.rollback()
                        print(f"skip schema {f.name}: {e.pgerror.strip()}")
                        continue
                    raise


def to_ts(value: Any) -> datetime | None:
    if value is None:
        return None
    try:
        return datetime.fromtimestamp(int(value), tz=timezone.utc)
    except Exception:
        return None


def fetch_all(sqlite_conn: sqlite3.Connection, query: str) -> List[sqlite3.Row]:
    cur = sqlite_conn.execute(query)
    return cur.fetchall()


def execute_values(pg_conn, table: str, cols: Sequence[str], rows: Iterable[Sequence[Any]], page_size: int) -> int:
    rows_list = list(rows)
    if not rows_list:
        return 0
    placeholders = "(" + ",".join(["%s"] * len(cols)) + ")"
    query = f"INSERT INTO {table} ({', '.join(cols)}) VALUES %s ON CONFLICT DO NOTHING"
    with pg_conn, pg_conn.cursor() as cur:
        extras.execute_values(cur, query, rows_list, template=placeholders, page_size=page_size)
    return len(rows_list)


def migrate_table_data(sqlite_conn: sqlite3.Connection, pg_conn, batch_size: int, include_saved_msgs: bool) -> None:
    total = 0

    def maybe(name: str, fn):
        nonlocal total
        if not table_exists(sqlite_conn, name):
            print(f"skip {name}: table not found in source")
            return
        total += fn()

    maybe(
        "users",
        lambda: execute_values(
            pg_conn,
            "users",
            [
                "id",
                "updated_at",
                "user_id",
                "first_name",
                "last_name",
                "profile_update_at",
                "profile_photo",
                "timezone",
            ],
            (
                (
                    r["id"],
                    to_ts(r["updated_at"]),
                    r["user_id"],
                    r["first_name"],
                    r["last_name"],
                    to_ts(r["profile_update_at"]),
                    r["profile_photo"],
                    r["timezone"],
                )
                for r in fetch_all(
                    sqlite_conn,
                    """
                    SELECT id, updated_at, user_id, first_name, last_name, profile_update_at, profile_photo, timezone
                    FROM users
                    """,
                )
            ),
            batch_size,
        ),
    )

    maybe(
        "prpr_caches",
        lambda: execute_values(
            pg_conn,
            "prpr_caches",
            ["profile_photo_uid", "prpr_file_id"],
            ((r["profile_photo_uid"], r["prpr_file_id"]) for r in fetch_all(sqlite_conn, "SELECT profile_photo_uid, prpr_file_id FROM prpr_caches")),
            batch_size,
        ),
    )

    maybe(
        "chat_cfg",
        lambda: execute_values(
            pg_conn,
            "chat_cfg",
            [
                "id",
                "web_id",
                "auto_cvt_bili",
                "auto_ocr",
                "auto_calculate",
                "auto_exchange",
                "auto_check_adult",
                "save_messages",
                "enable_coc",
                "resp_nsfw_msg",
                "timezone",
            ],
            (
                (
                    r["id"],
                    r["web_id"],
                    bool(r["auto_cvt_bili"]),
                    bool(r["auto_ocr"]),
                    bool(r["auto_calculate"]),
                    bool(r["auto_exchange"]),
                    bool(r["auto_check_adult"]),
                    bool(r["save_messages"]),
                    bool(r["enable_coc"]),
                    bool(r["resp_nsfw_msg"]),
                    r["timezone"],
                )
                for r in fetch_all(
                    sqlite_conn,
                    """
                    SELECT id, web_id, auto_cvt_bili, auto_ocr, auto_calculate, auto_exchange,
                           auto_check_adult, save_messages, enable_coc, resp_nsfw_msg, timezone
                    FROM chat_cfg
                    """,
                )
            ),
            batch_size,
        ),
    )

    maybe(
        "chat_stat_daily",
        lambda: execute_values(
            pg_conn,
            "chat_stat_daily",
            [
                "chat_id",
                "stat_date",
                "message_count",
                "photo_count",
                "video_count",
                "sticker_count",
                "forward_count",
                "mars_count",
                "max_mars_count",
                "racy_count",
                "adult_count",
                "download_video_count",
                "download_audio_count",
                "dio_add_user_count",
                "dio_ban_user_count",
                "user_msg_stat",
                "msg_count_by_time",
                "msg_id_at_time_start",
            ],
            (
                (
                    r["chat_id"],
                    r["stat_date"],
                    r["message_count"],
                    r["photo_count"],
                    r["video_count"],
                    r["sticker_count"],
                    r["forward_count"],
                    r["mars_count"],
                    r["max_mars_count"],
                    r["racy_count"],
                    r["adult_count"],
                    r["download_video_count"],
                    r["download_audio_count"],
                    r["dio_add_user_count"],
                    r["dio_ban_user_count"],
                    r["user_msg_stat"],
                    r["msg_count_by_time"],
                    r["msg_id_at_time_start"],
                )
                for r in fetch_all(sqlite_conn, "SELECT * FROM chat_stat_daily")
            ),
            batch_size,
        ),
    )

    maybe(
        "saved_pics",
        lambda: execute_values(
            pg_conn,
            "saved_pics",
            ["file_uid", "file_id", "bot_rate", "rand_key", "user_rating_sum", "rate_user_count"],
            (
                (
                    r["file_uid"],
                    r["file_id"],
                    r["bot_rate"],
                    r["rand_key"],
                    r["user_rating_sum"],
                    r["rate_user_count"],
                )
                for r in fetch_all(
                    sqlite_conn,
                    "SELECT file_uid, file_id, bot_rate, rand_key, user_rating_sum, rate_user_count FROM saved_pics",
                )
            ),
            batch_size,
        ),
    )

    maybe(
        "saved_pics_rating",
        lambda: execute_values(
            pg_conn,
            "saved_pics_rating",
            ["file_uid", "user_id", "rating"],
            ((r["file_uid"], r["user_id"], r["rating"]) for r in fetch_all(sqlite_conn, "SELECT file_uid, user_id, rating FROM saved_pics_rating")),
            batch_size,
        ),
    )

    maybe(
        "pic_rate_counter",
        lambda: execute_values(
            pg_conn,
            "pic_rate_counter",
            ["rate", "count"],
            ((r["rate"], r["count"]) for r in fetch_all(sqlite_conn, "SELECT rate, count FROM pic_rate_counter")),
            batch_size,
        ),
    )

    maybe(
        "gemini_sessions",
        lambda: execute_values(
            pg_conn,
            "gemini_sessions",
            ["id", "chat_id", "chat_name", "chat_type"],
            ((r["id"], r["chat_id"], r["chat_name"], r["chat_type"]) for r in fetch_all(sqlite_conn, "SELECT id, chat_id, chat_name, chat_type FROM gemini_sessions")),
            batch_size,
        ),
    )

    maybe(
        "gemini_contents",
        lambda: execute_values(
            pg_conn,
            "gemini_contents",
            [
                "session_id",
                "chat_id",
                "msg_id",
                "role",
                "sent_time",
                "username",
                "msg_type",
                "reply_to_msg_id",
                "text",
                "blob",
                "mime_type",
                "quote_part",
                "thought_signature",
            ],
            (
                (
                    r["session_id"],
                    r["chat_id"],
                    r["msg_id"],
                    r["role"],
                    to_ts(r["sent_time"]),
                    r["username"],
                    r["msg_type"],
                    r["reply_to_msg_id"],
                    r["text"],
                    r["blob"],
                    r["mime_type"],
                    r["quote_part"],
                    r["thought_signature"],
                )
                for r in fetch_all(
                    sqlite_conn,
                    """
                    SELECT session_id, chat_id, msg_id, role, sent_time, username, msg_type,
                           reply_to_msg_id, text, blob, mime_type, quote_part, thought_signature
                    FROM gemini_contents
                    """,
                )
            ),
            batch_size,
        ),
    )

    maybe(
        "bili_inline_results",
        lambda: execute_values(
            pg_conn,
            "bili_inline_results",
            ["uid", "text", "chat_id", "msg_id"],
            ((r["uid"], r["text"], r["chat_id"], r["msg_id"]) for r in fetch_all(sqlite_conn, "SELECT uid, text, chat_id, msg_id FROM bili_inline_results")),
            batch_size,
        ),
    )

    maybe(
        "yt_dl_results",
        lambda: execute_values(
            pg_conn,
            "yt_dl_results",
            ["url", "audio_only", "resolution", "file_id", "title", "description", "uploader", "upload_count"],
            (
                (
                    r["url"],
                    bool(r["audio_only"]),
                    r["resolution"],
                    r["file_id"],
                    r["title"],
                    r["description"],
                    r["uploader"],
                    r["upload_count"],
                )
                for r in fetch_all(
                    sqlite_conn,
                    """
                    SELECT url, audio_only, resolution, file_id, title, description, uploader, upload_count
                    FROM yt_dl_results
                    """,
                )
            ),
            batch_size,
        ),
    )

    if include_saved_msgs:
        if not table_exists(sqlite_conn, "saved_msgs"):
            print("saved_msgs table not found in source; skipping saved messages.")
        else:
            total += execute_values(
                pg_conn,
                "saved_msgs",
                [
                    "message_id",
                    "chat_id",
                    "from_user_id",
                    "sender_chat_id",
                    "date",
                    "forward_origin_name",
                    "forward_origin_id",
                    "message_thread_id",
                    "reply_to_message_id",
                    "reply_to_chat_id",
                    "via_bot_id",
                    "edit_date",
                    "media_group_id",
                    "text",
                    "entities_json",
                    "media_id",
                    "media_uid",
                    "media_type",
                    "extra_data",
                    "extra_type",
                ],
                (
                    (
                        r["message_id"],
                        r["chat_id"],
                        r["from_user_id"],
                        r["sender_chat_id"],
                        to_ts(r["date"]),
                        r["forward_origin_name"],
                        r["forward_origin_id"],
                        r["message_thread_id"],
                        r["reply_to_message_id"],
                        r["reply_to_chat_id"],
                        r["via_bot_id"],
                        to_ts(r["edit_date"]),
                        r["media_group_id"],
                        r["text"],
                        r["entities_json"],
                        r["media_id"],
                        r["media_uid"],
                        r["media_type"],
                        r["extra_data"],
                        r["extra_type"],
                    )
                    for r in fetch_all(
                        sqlite_conn,
                        """
                        SELECT message_id, chat_id, from_user_id, sender_chat_id, date, forward_origin_name, forward_origin_id,
                               message_thread_id, reply_to_message_id, reply_to_chat_id, via_bot_id, edit_date, media_group_id,
                               text, entities_json, media_id, media_uid, media_type, extra_data, extra_type
                        FROM saved_msgs
                        """,
                    )
                ),
                batch_size,
            )

            if table_exists(sqlite_conn, "raw_update"):
                total += execute_values(
                    pg_conn,
                    "raw_update",
                    ["id", "chat_id", "message_id", "raw_update"],
                    ((r["id"], r["chat_id"], r["message_id"], r["raw_update"]) for r in fetch_all(sqlite_conn, "SELECT id, chat_id, message_id, raw_update FROM raw_update")),
                    batch_size,
                )

            if table_exists(sqlite_conn, "edit_history"):
                total += execute_values(
                    pg_conn,
                    "edit_history",
                    ["chat_id", "message_id", "edit_id", "text"],
                    ((r["chat_id"], r["message_id"], r["edit_id"], r["text"]) for r in fetch_all(sqlite_conn, "SELECT chat_id, message_id, edit_id, text FROM edit_history")),
                    batch_size,
                )

    print(f"Migration complete, inserted/attempted rows: {total}")


def bump_sequences(pg_conn) -> None:
    seq_tables: Sequence[Tuple[str, str]] = [
        ("users", "id"),
        ("bili_inline_results", "uid"),
        ("gemini_sessions", "id"),
        ("raw_update", "id"),
    ]
    with pg_conn, pg_conn.cursor() as cur:
        for table, col in seq_tables:
            cur.execute(
                f"SELECT setval(pg_get_serial_sequence(%s, %s), COALESCE(MAX({col}), 1)) FROM {table}",
                (table, col),
            )


def main() -> None:
    args = parse_args()
    pg_url = load_pg_url(args)
    sqlite_conn = connect_sqlite(args.sqlite_path)
    pg_conn = connect_pg(pg_url)
    try:
        exec_schema(pg_conn, args.include_saved_msgs)
        migrate_table_data(sqlite_conn, pg_conn, args.batch_size, args.include_saved_msgs)
        bump_sequences(pg_conn)
    finally:
        sqlite_conn.close()
        pg_conn.close()


if __name__ == "__main__":
    main()
