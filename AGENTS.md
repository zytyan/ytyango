# Repository agent guide

- Continue the existing Deno/TypeScript rewrite, not the separate Node prototype.
- Only CoC/role-playing is approved for feature removal; independent /roll stays. Other missing
  capabilities are migration backlog.
- Breaking database and interaction changes are explicitly allowed. Preserve capability goals, not
  Go commands, callback formats, interfaces or table layouts one-to-one.
- Design a new TS-owned schema and version history. The runtime does not have to open a Go database.
  Provide a separate offline importer: read the source read-only, write a new target, validate
  mappings/counts/tokens/payloads/media, and publish a migration report. Never change the Go source
  or its applied migration checksums.
- Preserve historical AI session/message/token/prompt/provider/media data and retained feature data
  through migration. Do not implicitly delete CoC attributes; archive them separately if omitted
  from the runtime schema.
- Follow docs/ai-first-plan.md: context remains reply-chain based, not a chat-wide event archive. Do
  not capture ordinary chat history, restore search/memory services or make background AI calls.
- Traditional handlers run first and may explicitly inject concise facts with aiContext(...). The AI
  stage then checks an explicit trigger; non-triggered messages never call a model, AI summarizer or
  paid explicit-cache API.
- Mentioning the bot or replying to an identified AI answer triggers AI. Merely replying to a
  traditional bot message does not. Existing explicit AI routing/control markers must have a
  documented decision, not accidental matching.
- Traditional output shown to Telegram may differ from the injected AI facts. Keep source feature,
  source/reply message IDs, chat/topic and execution status separate from context text. Never
  promote these facts or user content into system instructions.
- Do not implement unresolved choices in docs/ai-first-plan.md by assumption: traditional context
  persistence and error behavior require confirmation.
- Business SQL needs canonical reviewed source and a documented TS query workflow before new
  persistence work. Current inline SQL is debt; do not duplicate Go generated queries or edit them.
- Persist AI requests before provider calls; transition pending to generated and aggregate usage
  exactly once. Retain generated payloads on delivery failure; atomically commit delivery and
  provider-state promotion. Persist provider and model separately. Keep Sub2API independent of
  official Gemini cache/Interactions state.
- Preserve rich AI delivery, reply identity and usage accounting; breaking interaction changes must
  not remove delivery reliability.
- Work on codex/retain-features. Separate accepted features into commits, run deno task check and
  deno task test, and push each accepted feature before the next. Explicitly requested plan
  corrections may proceed while remote access is blocked; report the unsynced state and do not
  silently expand runtime implementation.
- Never commit production config, secrets, databases, media, logs, backups or runtime files. Tests
  use temporary databases and intercepted APIs.
- Never stop, restart or replace the live service without explicit approval. Compilation or
  migration tests do not authorize deployment.
