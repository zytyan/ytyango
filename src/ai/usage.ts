function rawResponse(value: unknown): Record<string, unknown> | undefined {
  try {
    const text = typeof value === "string"
      ? value
      : value instanceof Uint8Array
      ? new TextDecoder().decode(value)
      : "";
    const parsed = JSON.parse(text);
    return parsed && typeof parsed === "object" ? parsed : undefined;
  } catch {
    return undefined;
  }
}

/** Stored SDK totals are authoritative here; raw diagnostics never replace them. */
export function formatUsage(usage: Record<string, unknown>): string {
  const value = (name: string) => Number(usage[name] ?? 0);
  const raw = rawResponse(usage.raw_payload);
  const native = raw?.usageMetadata as Record<string, unknown> | undefined;
  const other = raw?.usage as Record<string, unknown> | undefined;
  const reasoning = native?.thoughtsTokenCount ?? other?.total_thought_tokens ??
    (other?.completion_tokens_details as Record<string, unknown> | undefined)?.reasoning_tokens;
  const output = value("output_tokens");
  const details = typeof reasoning === "number" && reasoning >= 0 && reasoning <= output
    ? `（正文${output - reasoning}+推理${reasoning}）`
    : "（含推理）";
  const protocol = native ? "原生" : other && "total_input_tokens" in other ? "Interactions" : "";
  const model = [...String(usage.model)].slice(0, 35).join("");
  const lines = [
    `ID: ${value("session_id")} / ${model}`,
    `输入（接口上报）: ${value("input_tokens")} Token`,
    `输出: ${output}${details} Token`,
    `缓存（输入内）: ${value("cached_input_tokens")} Token`,
  ];
  const reference = other?.raw_prompt_token;
  const optional = [
    ...(protocol ? [`协议: ${protocol}`] : []),
    ...(typeof reference === "number" && reference !== value("input_tokens")
      ? [`原始输入参考: ${reference}（口径未确认）`]
      : []),
    ...(typeof native?.toolUsePromptTokenCount === "number" && native.toolUsePromptTokenCount > 0
      ? [`输入含工具: ${native.toolUsePromptTokenCount} Token`]
      : []),
    `消息: ${value("input_message_count")}（${value("input_first_msg_id")}-${
      value("input_last_msg_id")
    }）`,
    `缓存到期: ${
      value("cache_expire_at")
        ? new Date(value("cache_expire_at") * 1000).toLocaleString("zh-CN", {
          timeZone: "Asia/Shanghai",
        })
        : "无显式缓存"
    }`,
  ];
  for (const line of optional) {
    if ([...lines.join("\n"), ...line].length + 1 <= 200) lines.push(line);
  }
  return [...lines.join("\n")].slice(0, 200).join("");
}
