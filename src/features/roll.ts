import { randomInt } from "node:crypto";
import { type Bot, InlineKeyboard } from "grammy";

export const DELETE_CALLBACK = "delme";
const MAX_RANGE_WIDTH = 2 ** 48 - 1;

function integerOrDefault(text: string | undefined, fallback: number): number {
  if (text === undefined || !/^[+-]?\d+$/.test(text)) return fallback;
  const value = Number(text);
  if (!Number.isSafeInteger(value)) throw new Error("随机数范围超出安全整数限制");
  return value;
}

export function parseRollRange(args: readonly string[]): readonly [number, number] {
  let start = 1;
  let end = 20;
  if (args.length === 1) end = integerOrDefault(args[0], end);
  else if (args.length >= 2) {
    start = integerOrDefault(args[0], start);
    end = integerOrDefault(args[1], end);
    if (start > end) [start, end] = [end, start];
  }
  const width = end - start + 1;
  if (!Number.isSafeInteger(width) || width < 1 || width > MAX_RANGE_WIDTH) {
    throw new Error("随机数范围无效或过大");
  }
  return [start, end];
}

export function roll(
  args: readonly string[],
  sample: (exclusiveMaximum: number) => number = (maximum) => randomInt(maximum),
): string {
  const [start, end] = parseRollRange(args);
  const offset = sample(end - start + 1);
  if (!Number.isInteger(offset) || offset < 0 || offset > end - start) {
    throw new Error("Invalid random sample");
  }
  return `在${start}-${end}的roll点中，你掷出了${start + offset}`;
}

export function registerRoll(bot: Bot): void {
  bot.command("roll", async (ctx) => {
    const args = ctx.match.trim() === "" ? [] : ctx.match.trim().split(/\s+/u);
    let text: string;
    try {
      text = roll(args);
    } catch {
      await ctx.reply("随机数范围无效或过大", {
        reply_parameters: { message_id: ctx.msg.message_id },
      });
      return;
    }
    await ctx.reply(text, {
      reply_parameters: { message_id: ctx.msg.message_id },
      reply_markup: new InlineKeyboard().text("删除该消息", DELETE_CALLBACK),
    });
  });

  bot.callbackQuery(DELETE_CALLBACK, async (ctx) => {
    await ctx.answerCallbackQuery();
    const message = ctx.callbackQuery.message;
    if (message === undefined) return;
    await ctx.api.deleteMessage(message.chat.id, message.message_id);
    if ("reply_to_message" in message && message.reply_to_message !== undefined) {
      await ctx.api.deleteMessage(message.chat.id, message.reply_to_message.message_id);
    }
  });
}
