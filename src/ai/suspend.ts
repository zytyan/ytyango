/** Undefined is invalid input, null is permanent, numbers are durations in seconds. */
export function parseSuspension(value: string): number | null | "off" | undefined {
  if (!value) return null;
  if (value === "off") return "off";
  const match = value.match(/^([1-9]\d*)([smhd])$/);
  if (!match) return undefined;
  const seconds = Number(match[1]) * ({ s: 1, m: 60, h: 3600, d: 86400 }[match[2]!] ?? 0);
  return Number.isSafeInteger(seconds) ? seconds : undefined;
}
