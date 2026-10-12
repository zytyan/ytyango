import { SQL } from "../db/queries.ts";
import type { MainDatabase } from "../db/database.ts";

export interface ChatConfig {
  readonly id: number;
  readonly autoCvtBili: boolean;
  readonly autoCalculate: boolean;
  readonly autoExchange: boolean;
  readonly autoCheckAdult: boolean;
  readonly respNsfwMsg: boolean;
  readonly timezone: number;
  readonly inDatabase: boolean;
}

export type ConfigField =
  | "auto_cvt_bili"
  | "auto_calculate"
  | "auto_exchange"
  | "auto_check_adult"
  | "resp_nsfw_msg";

interface UserStat {
  messageCount: number;
  messageLength: number;
}

interface Snapshot {
  chatId: number;
  statDate: number;
  timezone: number;
  messageCount: number;
  photoCount: number;
  videoCount: number;
  stickerCount: number;
  forwardCount: number;
  marsCount: number;
  maxMarsCount: number;
  racyCount: number;
  adultCount: number;
  downloadVideoCount: number;
  downloadAudioCount: number;
  dioAddUserCount: number;
  dioBanUserCount: number;
  users: Map<number, UserStat>;
  buckets: number[];
  firstMessageIds: number[];
  dirty: boolean;
}

export interface StatisticalMessage {
  readonly chatId: number;
  readonly messageId: number;
  readonly sentAt: number;
  readonly user: {
    readonly id: number;
    readonly firstName: string;
    readonly lastName?: string;
    readonly username?: string;
  };
  readonly text: string;
  readonly photo: boolean;
  readonly video: boolean;
  readonly sticker: boolean;
  readonly forwarded: boolean;
}

const segmenter = new Intl.Segmenter(undefined, { granularity: "grapheme" });

function graphemeLength(text: string): number {
  let count = 0;
  for (const _ of segmenter.segment(text)) count++;
  return count;
}

function dateKey(unix: number, timezone: number): number {
  return Math.floor((unix + timezone) / 86_400);
}

function bucket(unix: number, timezone: number): number {
  const seconds = ((unix + timezone) % 86_400 + 86_400) % 86_400;
  return Math.floor(seconds / 600);
}

function escapeHtml(value: string): string {
  return value.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
}

export class StatsService {
  readonly #cache = new Map<string, Snapshot>();
  readonly #pending = new Map<string, Snapshot>();
  readonly #capacity = 512;

  constructor(readonly db: MainDatabase) {}

  config(chatId: number): ChatConfig {
    const row = this.db.get<{
      id: number;
      auto_cvt_bili: number;
      auto_calculate: number;
      auto_exchange: number;
      auto_check_adult: number;
      resp_nsfw_msg: number;
      timezone: number;
    }>(
      "SELECT id,auto_cvt_bili,auto_calculate,auto_exchange,auto_check_adult,resp_nsfw_msg,timezone FROM chat_cfg WHERE id=?",
      chatId,
    );
    return row
      ? {
        id: row.id,
        autoCvtBili: row.auto_cvt_bili === 1,
        autoCalculate: row.auto_calculate === 1,
        autoExchange: row.auto_exchange === 1,
        autoCheckAdult: row.auto_check_adult === 1,
        respNsfwMsg: row.resp_nsfw_msg === 1,
        timezone: row.timezone,
        inDatabase: true,
      }
      : {
        id: chatId,
        autoCvtBili: false,
        autoCalculate: false,
        autoExchange: false,
        autoCheckAdult: false,
        respNsfwMsg: false,
        timezone: 8 * 3600,
        inDatabase: false,
      };
  }

  setConfigField(
    chatId: number,
    field: ConfigField,
    value: boolean,
  ): ChatConfig {
    const current = this.config(chatId);
    const values: Record<ConfigField, boolean> = {
      auto_cvt_bili: current.autoCvtBili,
      auto_calculate: current.autoCalculate,
      auto_exchange: current.autoExchange,
      auto_check_adult: current.autoCheckAdult,
      resp_nsfw_msg: current.respNsfwMsg,
    };
    values[field] = value;
    this.db.transaction(() => {
      this.db.run(SQL.EnsureChatConfig, chatId, current.timezone);
      this.db.run(SQL.SetChatConfig, ...Object.values(values).map((v) => v ? 1 : 0), chatId);
    });
    return this.config(chatId);
  }

  record(message: StatisticalMessage): void {
    this.db.run(
      `INSERT INTO users(user_id,updated_at,first_name,last_name,username) VALUES(?,?,?,?,?)
ON CONFLICT(user_id) DO UPDATE SET updated_at=excluded.updated_at,first_name=excluded.first_name,
last_name=excluded.last_name,username=excluded.username`,
      message.user.id,
      message.sentAt,
      message.user.firstName,
      message.user.lastName ?? null,
      message.user.username ?? null,
    );
    const stat = this.#snapshot(message.chatId, message.sentAt);
    stat.messageCount++;
    if (message.photo) stat.photoCount++;
    if (message.video) stat.videoCount++;
    if (message.sticker) stat.stickerCount++;
    if (message.forwarded) stat.forwardCount++;
    const user = stat.users.get(message.user.id) ?? { messageCount: 0, messageLength: 0 };
    user.messageCount++;
    user.messageLength += graphemeLength(message.text);
    stat.users.set(message.user.id, user);
    const index = bucket(message.sentAt, stat.timezone);
    stat.buckets[index] = (stat.buckets[index] ?? 0) + 1;
    if (!stat.firstMessageIds[index]) stat.firstMessageIds[index] = message.messageId;
    stat.dirty = true;
  }

  increment(
    chatId: number,
    kind: "racy" | "adult" | "download_video" | "download_audio",
    unix = Math.floor(Date.now() / 1000),
  ): void {
    const stat = this.#snapshot(chatId, unix);
    if (kind === "racy") stat.racyCount++;
    else if (kind === "adult") stat.adultCount++;
    else if (kind === "download_video") stat.downloadVideoCount++;
    else stat.downloadAudioCount++;
    stat.dirty = true;
  }

  flush(): void {
    let firstError: unknown;
    for (const [key, stat] of [...this.#cache, ...this.#pending]) {
      try {
        this.#save(stat);
        this.#pending.delete(key);
      } catch (error) {
        this.#pending.set(key, stat);
        firstError ??= error;
      }
    }
    if (firstError) throw firstError;
  }

  rank(chatId: number, unix = Math.floor(Date.now() / 1000)): string {
    const stat = this.#snapshot(chatId, unix);
    const sorted = [...stat.users].sort((a, b) => b[1].messageCount - a[1].messageCount);
    if (sorted.length === 0) return "没有数据";
    const lines = sorted.slice(0, 10).map(([userId, value]) =>
      `${this.#userName(userId)}: ${value.messageCount}`
    );
    if (sorted.length > 10) {
      lines.push(
        `其他人: ${sorted.slice(10).reduce((sum, item) => sum + item[1].messageCount, 0)}`,
      );
    }
    return lines.join("\n");
  }

  dailyMessage(chatId: number, targetUnix: number): string {
    const stat = this.#snapshot(chatId, targetUnix);
    this.#save(stat);
    const active = [...stat.users].sort((a, b) => b[1].messageCount - a[1].messageCount).slice(
      0,
      3,
    );
    const names = active.map(([id]) => this.#userName(id));
    const activeNames = names.length === 0
      ? "没有人"
      : names.length === 1
      ? names[0]!
      : `${names.slice(0, -1).join("、")}和${names.at(-1)}`;
    const mostActive = names[0] ?? "没有人";
    const maximum = Math.max(...stat.buckets, 0);
    const timeId = stat.buckets.indexOf(maximum);
    const activeTime = `${String(Math.floor(timeId / 6)).padStart(2, "0")}:${
      String((timeId % 6) * 10).padStart(2, "0")
    }`;
    const messageId = stat.firstMessageIds[timeId] ?? 0;
    const groupLinkId = -chatId - 1_000_000_000_000;
    const timeLink = messageId
      ? `<a href="https://t.me/c/${groupLinkId}/${messageId}">${activeTime}</a>`
      : activeTime;
    const date = new Date((stat.statDate * 86_400 - stat.timezone) * 1000);
    const dateText = new Intl.DateTimeFormat("zh-CN", {
      timeZone: "UTC",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).format(date).replaceAll("/", "年").replace(/年(\d{2})年/, "年$1月") + "日";
    return `早上好！吹水群！
今天是${dateText}，昨天的发言统计，最后的结果是满打满算的整整${stat.messageCount}条，你们这些家伙都不用上班的吗？
多亏了${escapeHtml(activeNames)}没完没了的摸鱼吹水，光${escapeHtml(mostActive)}一个人就发了${
      active[0]?.[1].messageCount ?? 0
    }条。但有一个晶哥也发话了，我看你们全都得喝茶，因为平子肯定咽不下这口气。
群里一共发了${stat.photoCount}张图片，还有${stat.stickerCount}个表情，又是只发表情的社恐干的好事。与此同时，毅力号还在火星上替火星人找到了${stat.marsCount}张图，火星次数最多的图让你们火星了${stat.maxMarsCount}次，真是一群火星人。
群里发了${
      stat.adultCount + stat.racyCount
    }张色图，里面还有${stat.adultCount}张R18，你们今天发色图，明天FBI就来敲你家门了。
智乃酱帮你们下了${stat.downloadVideoCount}个视频，${stat.downloadAudioCount}个音频，今日份的娱乐就到这里吧。
而群里最热闹的${timeLink}，这十分钟居然发了${maximum}条，好吧，吹水群还是那个吹水群。
我是你们的铁哥们智乃酱，和我一起开启完蛋操的新一天吧！`;
  }

  #snapshot(chatId: number, unix: number): Snapshot {
    const timezone = this.config(chatId).timezone;
    const day = dateKey(unix, timezone);
    const key = `${chatId}:${day}`;
    const found = this.#cache.get(key) ?? this.#pending.get(key);
    if (found) return found;
    if (this.#cache.size >= this.#capacity) {
      const oldest = this.#cache.entries().next().value as [string, Snapshot] | undefined;
      if (oldest) {
        this.#cache.delete(oldest[0]);
        try {
          this.#save(oldest[1]);
        } catch {
          this.#pending.set(oldest[0], oldest[1]);
        }
      }
    }
    const daily = this.db.get<Record<string, number>>(
      "SELECT * FROM chat_stat_daily WHERE chat_id=? AND stat_date=?",
      chatId,
      day,
    );
    if (!daily) {
      this.db.run("INSERT INTO chat_stat_daily(chat_id,stat_date) VALUES(?,?)", chatId, day);
    }
    const stat: Snapshot = {
      chatId,
      statDate: day,
      timezone,
      messageCount: daily?.message_count ?? 0,
      photoCount: daily?.photo_count ?? 0,
      videoCount: daily?.video_count ?? 0,
      stickerCount: daily?.sticker_count ?? 0,
      forwardCount: daily?.forward_count ?? 0,
      marsCount: daily?.mars_count ?? 0,
      maxMarsCount: daily?.max_mars_count ?? 0,
      racyCount: daily?.racy_count ?? 0,
      adultCount: daily?.adult_count ?? 0,
      downloadVideoCount: daily?.download_video_count ?? 0,
      downloadAudioCount: daily?.download_audio_count ?? 0,
      dioAddUserCount: daily?.dio_add_user_count ?? 0,
      dioBanUserCount: daily?.dio_ban_user_count ?? 0,
      users: new Map(
        this.db.all<{ user_id: number; message_count: number; message_length: number }>(
          "SELECT user_id,message_count,message_length FROM chat_stat_user_daily WHERE chat_id=? AND stat_date=?",
          chatId,
          day,
        ).map((
          item,
        ) => [item.user_id, {
          messageCount: item.message_count,
          messageLength: item.message_length,
        }]),
      ),
      buckets: Array(144).fill(0),
      firstMessageIds: Array(144).fill(0),
      dirty: false,
    };
    for (
      const item of this.db.all<{ bucket: number; message_count: number; first_msg_id: number }>(
        "SELECT bucket,message_count,first_msg_id FROM chat_stat_bucket_daily WHERE chat_id=? AND stat_date=? ORDER BY bucket",
        chatId,
        day,
      )
    ) {
      stat.buckets[item.bucket] = item.message_count;
      stat.firstMessageIds[item.bucket] = item.first_msg_id;
    }
    this.#cache.set(key, stat);
    return stat;
  }

  #save(stat: Snapshot): void {
    if (!stat.dirty) return;
    this.db.transaction(() => {
      this.db.run(
        `UPDATE chat_stat_daily SET message_count=?,photo_count=?,video_count=?,sticker_count=?,
forward_count=?,mars_count=?,max_mars_count=?,racy_count=?,adult_count=?,download_video_count=?,
download_audio_count=?,dio_add_user_count=?,dio_ban_user_count=? WHERE chat_id=? AND stat_date=?`,
        stat.messageCount,
        stat.photoCount,
        stat.videoCount,
        stat.stickerCount,
        stat.forwardCount,
        stat.marsCount,
        stat.maxMarsCount,
        stat.racyCount,
        stat.adultCount,
        stat.downloadVideoCount,
        stat.downloadAudioCount,
        stat.dioAddUserCount,
        stat.dioBanUserCount,
        stat.chatId,
        stat.statDate,
      );
      for (const [userId, value] of stat.users) {
        this.db.run(
          `INSERT INTO chat_stat_user_daily(chat_id,stat_date,user_id,message_count,message_length)
VALUES(?,?,?,?,?) ON CONFLICT(chat_id,stat_date,user_id) DO UPDATE SET
message_count=excluded.message_count,message_length=excluded.message_length`,
          stat.chatId,
          stat.statDate,
          userId,
          value.messageCount,
          value.messageLength,
        );
      }
      for (let index = 0; index < 144; index++) {
        this.db.run(
          `INSERT INTO chat_stat_bucket_daily(chat_id,stat_date,bucket,message_count,first_msg_id)
VALUES(?,?,?,?,?) ON CONFLICT(chat_id,stat_date,bucket) DO UPDATE SET
message_count=excluded.message_count,first_msg_id=excluded.first_msg_id`,
          stat.chatId,
          stat.statDate,
          index,
          stat.buckets[index] ?? 0,
          stat.firstMessageIds[index] ?? 0,
        );
      }
    });
    stat.dirty = false;
  }

  #userName(userId: number): string {
    const user = this.db.get<{ first_name: string; last_name: string | null }>(
      "SELECT first_name,last_name FROM users WHERE user_id=?",
      userId,
    );
    return user
      ? `${user.first_name}${user.last_name ? ` ${user.last_name}` : ""}`
      : `用户${userId}`;
  }
}
