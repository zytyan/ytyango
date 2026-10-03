# Repository agent guide

- Continue the existing Deno/TypeScript rewrite; do not replace it with the separate Node prototype.
- Only the CoC/role-playing module is approved for feature removal. Independent /roll stays. Missing
  features in docs/features.md are migration backlog, not approved cuts.
- Do not reintroduce retired AI memory, message archives or search services. Do not add new product
  features during migration.
- Preserve all historical AI session/message/token/prompt/provider/media data and retained feature
  data. Removing CoC does not authorize deleting its historical attributes.
- Work on codex/retain-features. Keep independently accepted features in separate commits, run deno
  task check and deno task test, and push each accepted feature before starting the next. Report
  remote permission/creation failures instead of switching repositories or force-pushing.
- Keep Go's applied migration checksums immutable. The current TS V9 schema is not compatible with
  the current Go V11 schema. Never experiment on production data or silently migrate it at startup.
  Update docs when compatibility changes.
- Business SQL must have a canonical, reviewed source. Current inline SQL is migration debt;
  document the TS query workflow before introducing new business SQL. Leave Go offline migration
  tools authoritative until replacements have independent acceptance.
- AI requests must be persisted before provider calls; transition pending to generated and aggregate
  usage exactly once. Preserve payloads on delivery failure and commit delivery/provider-state
  promotion atomically. Persist provider and model separately. Keep Sub2API independent of official
  Gemini Interactions/cache state.
- Preserve AI Rich Markdown delivery, reply identity and usage keyboard. Do not replace it with
  ordinary Markdown.
- Never commit production config, tokens, databases, media, logs, backups or runtime files. Tests
  use temporary databases and intercepted provider/Telegram APIs; never live credentials.
- Never stop, restart or replace the live service without explicit approval. A successful compile is
  not approval to deploy.
