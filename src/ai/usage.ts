/** Keep the Go alert layout; display rounding never changes stored usage. */
export function formatUsage(usage: Record<string, unknown>): string {
  const value = (name: string) => Number(usage[name] ?? 0);
  const tokens = (name: string) => `${(value(name) / 1000).toFixed(2)} ktoken`;
  const count = value("input_message_count");
  const messages = count > 0
    ? `${count}（${value("input_first_msg_id")}-${value("input_last_msg_id")}）`
    : "未记录";
  const expiry = value("cache_expire_at");
  const date = expiry
    ? new Intl.DateTimeFormat("sv-SE", {
      timeZone: "Asia/Shanghai",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    }).format(new Date(expiry * 1000))
    : "无显式缓存";
  const text = `ID: ${value("session_id")}\n消息: ${messages}\n模型: ${
    String(usage.model ?? "")
  }\n输入: ${tokens("input_tokens")}\n输出: ${tokens("output_tokens")}\n缓存: ${
    tokens("cached_input_tokens")
  }\n缓存到期：${date}`;
  const characters = [...text];
  return characters.length <= 200 ? text : `${characters.slice(0, 199).join("")}…`;
}
