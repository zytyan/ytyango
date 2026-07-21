import { assertEquals } from "@std/assert";
import { parseConfig } from "../../src/config.ts";

Deno.test("Go-style YAML fields and defaults remain compatible", () => {
  const config = parseConfig(`
database-path: /srv/ytyan/main.db
bot-token: token
god: 42
ai-chats: [-1001]
content-moderator:
  endpoint: https://azure.example/
  api-key: secret
`);
  assertEquals(config.databasePath, "/srv/ytyan/main.db");
  assertEquals(config.aiMediaPath, "/srv/ytyan/ai-media");
  assertEquals(config.god, 42);
  assertEquals(config.aiChats, [-1001]);
  assertEquals(config.contentModerator.endpoint, "https://azure.example");
  assertEquals(config.telegramApiUrl, "https://api.telegram.org");
  assertEquals(config.deepSeekBaseUrl, "https://api.deepseek.com");
});
