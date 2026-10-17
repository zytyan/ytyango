/** DeepSeek occasionally wraps Anthropic search errors in a singleton result array. */
export function normalizeDeepseekSearchErrors(payload: unknown): boolean {
  if (
    !payload || typeof payload !== "object" || !("content" in payload) ||
    !Array.isArray(payload.content)
  ) return false;
  let changed = false;
  for (const block of payload.content) {
    if (
      !block || typeof block !== "object" || block.type !== "web_search_tool_result" ||
      !Array.isArray(block.content) || block.content.length !== 1
    ) continue;
    const error = block.content[0];
    if (
      !error || typeof error !== "object" || error.type !== "web_search_tool_result_error" ||
      typeof error.error_code !== "string"
    ) continue;
    block.content = error;
    changed = true;
  }
  return changed;
}
