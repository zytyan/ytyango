import type { Context } from "grammy";
import type { BotCommand, InlineKeyboardMarkup, Message } from "grammy/types";
import { topicId } from "./topic.ts";

type ReplyOptions = NonNullable<Parameters<Context["reply"]>[1]>;
const callbackTimes = new WeakMap<Context, number>();
export function recordCallbackReceived(ctx: Context): void {
  if (ctx.callbackQuery) callbackTimes.set(ctx, Date.now());
}
export type PrivateReplyOptions = ReplyOptions & {
  ephemeral_message_parameters?: { receiver_user_id: number; callback_query_id?: string };
};

export const PRIVATE_COMMANDS = new Set([
  "session_id",
  "session_help",
  "ai_reaction",
  "get_sysprompt",
  "sysprompt",
  "reset_sysprompt",
  "new_session",
  "show_usage",
  "model",
  "change_model",
  "chat_config",
]);

export function incomingPrivateCommand(message: Message): string {
  return (message.text ?? "").match(/^\/([a-z_]+)(?:@\w+)?(?:\s|$)/)?.[1] ?? "";
}

const commands: readonly (readonly [string, string])[] = [
  ["session_id", "查询 AI 会话 ID"],
  ["session_help", "会话使用帮助"],
  ["new_session", "重置本话题的默认 AI 会话"],
  ["model", "切换群或历史会话的模型"],
  ["change_model", "切换模型"],
  ["ai_reaction", "设置个人 AI 表情回应：on/off"],
  ["get_sysprompt", "查看本话题系统提示词"],
  ["sysprompt", "设置本话题系统提示词"],
  ["reset_sysprompt", "恢复本话题默认提示词"],
  ["show_usage", "切换全群 Token 用量按钮"],
  ["chat_config", "查看或修改全群设置"],
  ["roll", "掷骰子"],
  ["calc", "计算表达式"],
  ["ocr", "识别图片文字"],
  ["prpr", "生成头像舔图"],
  ["webp2png", "将静态 WebP 转为 PNG"],
  ["score", "图片审核评分"],
  ["getrank", "今日发言排行"],
  ["diag_sendstat", "发送昨日群统计"],
  ["downloadvideo", "下载视频"],
  ["downloadaudio", "下载音频"],
];

export function commandMenu(group: boolean): BotCommand[] {
  return commands.map(([command, description]) => ({
    command,
    description,
    is_ephemeral: group && PRIVATE_COMMANDS.has(command),
  }));
}

/** A normal message is not a 15-second eligibility grant. Only bot admins can initiate privately. */
export async function privateReplyOptions(
  ctx: Context,
  now = Date.now(),
): Promise<PrivateReplyOptions | undefined> {
  const message = ctx.message;
  const source = message ?? ctx.callbackQuery?.message;
  const normal: ReplyOptions = {
    ...(source?.message_id ? { reply_parameters: { message_id: source.message_id } } : {}),
    ...(topicId(source) ? { message_thread_id: topicId(source) } : {}),
  };
  if (!ctx.chat || !ctx.from || !["group", "supergroup"].includes(ctx.chat.type)) return normal;
  if (ctx.from.is_bot) return message?.ephemeral_message_id ? undefined : normal;
  const ephemeral = { receiver_user_id: ctx.from.id };
  if (ctx.callbackQuery && now - (callbackTimes.get(ctx) ?? now) < 15_000) {
    return {
      ephemeral_message_parameters: { ...ephemeral, callback_query_id: ctx.callbackQuery.id },
      ...(topicId(ctx.callbackQuery.message)
        ? { message_thread_id: topicId(ctx.callbackQuery.message) }
        : {}),
    };
  }
  if (message?.ephemeral_message_id && now - message.date * 1000 < 15_000) {
    return {
      ephemeral_message_parameters: ephemeral,
      reply_parameters: { ephemeral_message_id: message.ephemeral_message_id },
      ...(topicId(message) ? { message_thread_id: topicId(message) } : {}),
    };
  }
  let admin = false;
  try {
    const member = await ctx.api.getChatMember(ctx.chat.id, ctx.me.id);
    admin = member.status === "administrator" || member.status === "creator";
  } catch {
    // A failed permissions lookup must not accidentally publish a private response.
    return undefined;
  }
  const callbackMessage = ctx.callbackQuery?.message;
  const privateInput = message?.ephemeral_message_id ||
    (callbackMessage && "ephemeral_message_id" in callbackMessage &&
      callbackMessage.ephemeral_message_id);
  if (!admin) return privateInput ? undefined : normal;
  // Do not reply to an expired ephemeral ID, even when the bot can initiate privately.
  return {
    ...normal,
    ephemeral_message_parameters: ephemeral,
  };
}

/** Never retry an ambiguous private send or silently publish a failed private response. */
export async function replyPrivately(ctx: Context, text: string, options: ReplyOptions = {}) {
  const target = await privateReplyOptions(ctx);
  if (!target || !ctx.chat) return undefined;
  return await ctx.api.sendMessage(ctx.chat.id, text, {
    ...options,
    ...target,
    reply_parameters: target.reply_parameters,
  });
}

export async function editMenuText(
  ctx: Context,
  text: string,
  markup: InlineKeyboardMarkup,
): Promise<void> {
  const message = ctx.callbackQuery?.message;
  if (message && "ephemeral_message_id" in message && message.ephemeral_message_id) {
    await ctx.api.editEphemeralMessageText(
      message.chat.id,
      ctx.from!.id,
      message.ephemeral_message_id,
      text,
      { reply_markup: markup },
    );
  } else {
    await ctx.editMessageText(text, { reply_markup: markup });
  }
}

export async function editMenuMarkup(ctx: Context, markup: InlineKeyboardMarkup): Promise<void> {
  const message = ctx.callbackQuery?.message;
  if (message && "ephemeral_message_id" in message && message.ephemeral_message_id) {
    await ctx.api.editEphemeralMessageReplyMarkup(
      message.chat.id,
      ctx.from!.id,
      message.ephemeral_message_id,
      { reply_markup: markup },
    );
  } else {
    await ctx.editMessageReplyMarkup({ reply_markup: markup });
  }
}
