import type { AppConfig } from "../config.ts";
import type { MainDatabase } from "../db/database.ts";
import { BoundedCache, KeyedMutex } from "../shared/concurrency.ts";
import type { SentMessage, TelegramPort } from "../shared/types.ts";
import { normalizeModerationImage } from "../features/images.ts";

export interface NsfwPhoto {
  readonly fileId: string;
  readonly fileUniqueId: string;
  readonly width: number;
  readonly height: number;
}

export interface NsfwMessage {
  readonly chatId: number;
  readonly messageId: number;
  readonly mediaGroupId?: string;
  readonly hasMediaSpoiler: boolean;
  readonly photo: NsfwPhoto;
}

interface Detection {
  readonly severity: number;
  readonly animeIllustration: boolean;
}

const severityCriteria = {
  "0":
    "Safe: benign family or romantic imagery, everyday clothing and objects, nonsexual poses or activities, nonsexual nudity, animal reproduction, and sexual health contexts; ordinary hugs and nonsexual affectionate contact.",
  "2":
    "Low: suggestive behavior or presentation, fashion poses, body decoration, artistic nudity, and mildly provocative displays.",
  "4":
    "Medium: erotic commercial or amateur material; nude or partly clothed subjects intended to arouse, without visibly photographed penetration; erotic posing, less explicit artistic nudity, adult businesses, partially hidden sexual activity, backstage productions and pole performances.",
  "6":
    "High: clearly detailed sexual anatomy or sexual acts; pornography, sexual solicitation, voyeurism, sexual acts in art, nonconsensual sexual activity, exploitation or grooming of children, incest, sexual activity involving animals or deceased persons.",
};

export interface RatingResult {
  readonly existed: boolean;
  readonly oldRating: number;
  readonly changed: boolean;
}

export class NsfwService {
  readonly #cache = new BoundedCache<string, Detection>(500, 6 * 3600_000);
  readonly #groups = new KeyedMutex();
  readonly #rateLimit = new KeyedMutex();
  readonly #groupReplied = new Map<string, number>();
  #nextRequestAt = 0;

  constructor(
    readonly db: MainDatabase,
    readonly telegram: TelegramPort,
    readonly config: () => AppConfig,
    readonly incrementStat: (chatId: number, kind: "racy" | "adult") => void,
    readonly fetcher: typeof fetch = fetch,
    readonly normalize: (bytes: Uint8Array) => Promise<Uint8Array> = normalizeModerationImage,
  ) {}

  async detectAndReply(
    message: NsfwMessage,
    onReply?: (sent: SentMessage, result: Detection) => void,
  ): Promise<number> {
    const key = message.mediaGroupId
      ? `${message.chatId}:${message.mediaGroupId}`
      : `${message.chatId}:${message.messageId}`;
    return await this.#groups.use(key, async () => {
      this.#expireGroups();
      const result = await this.moderate(message.photo);
      const severity = result.severity;
      this.collect(message.photo, result);
      if (severity < 2 || severity > 7) return severity;
      this.incrementStat(message.chatId, severity >= 6 ? "adult" : "racy");
      if (message.mediaGroupId && this.#groupReplied.has(key)) return severity;
      const replies = message.hasMediaSpoiler
        ? ["给bot也看看~", "悄悄看一眼~", "不敢看~"]
        : ["涩涩的~", "好色哦~", "口夷~"];
      const sent = await this.telegram.sendMessage(
        message.chatId,
        replies[Math.floor(severity / 2) - 1]!,
        {
          reply_parameters: { message_id: message.messageId },
          reply_markup: this.ratingKeyboard(message.photo.fileUniqueId),
        },
      );
      if (message.mediaGroupId) this.#groupReplied.set(key, Date.now() + 10_000);
      onReply?.(sent, result);
      return severity;
    });
  }

  async score(photo: NsfwPhoto): Promise<number> {
    return (await this.moderate(photo)).severity;
  }

  collect(photo: NsfwPhoto, result: Detection) {
    if (result.severity >= 2 || result.animeIllustration) {
      this.savePicture(photo.fileUniqueId, photo.fileId, result.severity);
    }
    return this.getPicture(photo.fileUniqueId);
  }

  async moderate(photo: NsfwPhoto): Promise<Detection> {
    if (photo.width < 128 || photo.height < 128) throw new Error("image too small");
    const cached = this.#cache.get(photo.fileId);
    if (cached) return cached;
    return await this.#rateLimit.use(
      "cloudflare",
      () => latencyStage("moderation", () => this.#scoreUncached(photo)),
      "queue.moderation",
    );
  }

  async #scoreUncached(photo: NsfwPhoto): Promise<Detection> {
    const cached = this.#cache.get(photo.fileId);
    if (cached) return cached;
    const wait = Math.max(0, this.#nextRequestAt - Date.now());
    if (wait > 0) await new Promise((resolve) => setTimeout(resolve, wait));
    this.#nextRequestAt = Date.now() + 200;
    const config = this.config().contentModerator;
    if (!config.accountId || !config.apiToken) throw new Error("Cloudflare Workers AI 未配置");
    const file = await this.telegram.getFile(photo.fileId);
    const bytes = await this.normalize(
      await this.telegram.downloadFile(file, AbortSignal.timeout(10_000)),
    );
    let binary = "";
    for (let offset = 0; offset < bytes.length; offset += 0x8000) {
      binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
    }
    const response = await this.fetcher(
      `https://api.cloudflare.com/client/v4/accounts/${
        encodeURIComponent(config.accountId)
      }/ai/run/@cf/cloudflare/clef-flash`,
      {
        method: "POST",
        headers: {
          "content-type": "application/json",
          "Authorization": `Bearer ${config.apiToken}`,
        },
        body: JSON.stringify({
          model: "clef-flash",
          state: "Evaluate the single attached image for content moderation.",
          images: [{ content_type: "image/jpeg", base64: btoa(binary) }],
          questions: {
            microsoft_severity: {
              type: "choice",
              instructions:
                "Assess the attached image using Microsoft Sexual IMAGE severity definitions; select exactly one severity: 0 Safe, 2 Low, 4 Medium, or 6 High, based on visible evidence and context, not source or filename.",
              criteria: severityCriteria,
            },
            anime_illustration: {
              type: "choice",
              instructions:
                "Is this a Japanese anime/manga-style (2D or 二次元) illustration? Classify visual style independently of sexual content. Ordinary photos, text/interface screenshots and other illustration styles do not qualify.",
              criteria: {
                yes:
                  "A drawn or digital illustration in Japanese anime or manga style, whether safe or sexual.",
                no: "Not anime/manga style or insufficient visual evidence.",
              },
            },
          },
        }),
        signal: AbortSignal.timeout(10_000),
      },
    );
    if (!response.ok) {
      throw new Error(`Cloudflare moderation HTTP ${response.status}`);
    }
    const text = await response.text();
    if (text.length > 1024 * 1024) throw new Error("moderation response too large");
    const body = JSON.parse(text) as {
      success: boolean;
      result?: { answers?: Record<string, { type: string; choice: string }> };
    };
    const severityAnswer = body.result?.answers?.microsoft_severity;
    const anime = body.result?.answers?.anime_illustration;
    if (
      !body.success || severityAnswer?.type !== "choice" ||
      !["0", "2", "4", "6"].includes(severityAnswer.choice) || anime?.type !== "choice" ||
      !["yes", "no"].includes(anime.choice)
    ) throw new Error("invalid Cloudflare moderation choices");
    const result = {
      severity: Number(severityAnswer.choice),
      animeIllustration: anime.choice === "yes",
    };
    this.#cache.set(photo.fileId, result);
    return result;
  }

  savePicture(fileUniqueId: string, fileId: string, severity: number): void {
    if (severity < -1 || severity > 7) throw new RangeError("invalid picture severity");
    for (let attempt = 0; attempt < 3; attempt++) {
      const randomKey = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
      try {
        this.db.run(
          `INSERT INTO saved_pics(file_uid,file_id,bot_rate,rand_key) VALUES(?,?,?,?)
ON CONFLICT(file_uid) DO UPDATE SET file_id=excluded.file_id,bot_rate=excluded.bot_rate`,
          fileUniqueId,
          fileId,
          severity,
          randomKey,
        );
        return;
      } catch (error) {
        if (!String(error).includes("UNIQUE")) throw error;
      }
    }
    throw new Error(`failed to insert picture after random key retries: ${fileUniqueId}`);
  }

  rate(fileUniqueId: string, userId: number, rating: number): RatingResult {
    if (![0, 2, 4, 6].includes(rating)) throw new RangeError("invalid user rating");
    const old = this.db.get<{ rating: number }>(
      "SELECT rating FROM saved_pics_rating WHERE file_uid=? AND user_id=?",
      fileUniqueId,
      userId,
    );
    if (!old) {
      this.db.run(
        "INSERT INTO saved_pics_rating(file_uid,user_id,rating) VALUES(?,?,?)",
        fileUniqueId,
        userId,
        rating,
      );
      return { existed: false, oldRating: 0, changed: true };
    }
    if (old.rating === rating) return { existed: true, oldRating: rating, changed: false };
    this.db.run(
      "UPDATE saved_pics_rating SET rating=? WHERE file_uid=? AND user_id=?",
      rating,
      fileUniqueId,
      userId,
    );
    return { existed: true, oldRating: old.rating, changed: true };
  }

  getPicture(
    fileUniqueId: string,
  ): { fileId: string; botRate: number; userRate: number; sum: number; count: number } | undefined {
    const row = this.db.get<{
      file_id: string;
      bot_rate: number;
      user_rate: number;
      user_rating_sum: number;
      rate_user_count: number;
    }>(
      "SELECT file_id,bot_rate,user_rate,user_rating_sum,rate_user_count FROM saved_pics WHERE file_uid=?",
      fileUniqueId,
    );
    return row
      ? {
        fileId: row.file_id,
        botRate: row.bot_rate,
        userRate: row.user_rate,
        sum: row.user_rating_sum,
        count: row.rate_user_count,
      }
      : undefined;
  }

  randomPicture(start = 2, end = 7): { fileId: string; fileUniqueId: string } | undefined {
    start = Math.max(0, Math.min(7, start));
    end = Math.max(0, Math.min(7, end));
    if (end < start) [start, end] = [end, start];
    const counts = this.db.all<{ rate: number; count: number }>(
      "SELECT rate,count FROM pic_rate_counter WHERE rate BETWEEN ? AND ? AND count>0 ORDER BY rate",
      start,
      end,
    );
    const total = counts.reduce((sum, item) => sum + item.count, 0);
    if (total <= 0) return undefined;
    let target = Math.floor(Math.random() * total);
    let rate = start;
    for (const item of counts) {
      if (target < item.count) {
        rate = item.rate;
        break;
      }
      target -= item.count;
    }
    const pivot = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    const row = this.db.get<{ file_uid: string; file_id: string }>(
      `SELECT file_uid,file_id FROM saved_pics WHERE user_rate=? AND rand_key>?
ORDER BY rand_key LIMIT 1`,
      rate,
      pivot,
    ) ?? this.db.get<{ file_uid: string; file_id: string }>(
      "SELECT file_uid,file_id FROM saved_pics WHERE user_rate=? ORDER BY rand_key LIMIT 1",
      rate,
    );
    return row ? { fileId: row.file_id, fileUniqueId: row.file_uid } : undefined;
  }

  ratingKeyboard(fileUniqueId: string, extra = ""): Record<string, unknown> {
    const counts = new Map(
      this.db.all<{ rating: number; count: number }>(
        "SELECT rating,COUNT(*) AS count FROM saved_pics_rating WHERE file_uid=? GROUP BY rating",
        fileUniqueId,
      ).map((item) => [item.rating, item.count]),
    );
    const label = (text: string, rating: number) =>
      counts.get(rating) ? `${text} (${counts.get(rating)} 用户评分)` : text;
    const data = (rating: number) => `nsfw:${rating}:${fileUniqueId}:${extra}`;
    return {
      inline_keyboard: [
        [{ text: label("不色！", 0), callback_data: data(0) }, {
          text: label("有点涩", 2),
          callback_data: data(2),
        }],
        [{ text: label("好色哦", 4), callback_data: data(4) }, {
          text: label("色爆了", 6),
          callback_data: data(6),
        }],
      ],
    };
  }

  #expireGroups(): void {
    const now = Date.now();
    for (const [key, expiry] of this.#groupReplied) {
      if (expiry <= now) this.#groupReplied.delete(key);
    }
  }
}
import { latencyStage } from "../shared/latency.ts";
