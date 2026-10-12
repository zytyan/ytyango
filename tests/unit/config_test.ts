import { assertEquals } from "@std/assert";
import { parseConfig } from "../../src/config.ts";

Deno.test("Go-style YAML fields and defaults remain compatible", () => {
  const config = parseConfig(`
database-path: /srv/ytyan/main.db
bot-token: token
god: 42
ai-chats: [-1001]
content-moderator:
  account-id: cloudflare-account
  api-token: secret
ocr:
  endpoint: https://azure.example/
  api-key: ocr-secret
`);
  assertEquals(config.databasePath, "/srv/ytyan/main.db");
  assertEquals(config.aiMediaPath, "/srv/ytyan/ai-media");
  assertEquals(config.god, 42);
  assertEquals(config.aiChats, [-1001]);
  assertEquals(config.contentModerator.accountId, "cloudflare-account");
  assertEquals(config.telegramApiUrl, "https://api.telegram.org");
  assertEquals(config.deepSeekBaseUrl, "https://api.deepseek.com");
  assertEquals(config.sub2apiKey, "");
  assertEquals(config.sub2apiBaseUrl, "http://127.0.0.1:18080/antigravity/v1beta");
});

Deno.test("Sub2API native base URL and credential are independently configured", () => {
  const config = parseConfig(`database-path: /tmp/test.db
gemini-key: official-key
sub2api-key: proxy-key
sub2api-base-url: https://proxy.example/antigravity/v1beta/
`);
  assertEquals(config.geminiKey, "official-key");
  assertEquals(config.sub2apiKey, "proxy-key");
  assertEquals(config.sub2apiBaseUrl, "https://proxy.example/antigravity/v1beta");
});
