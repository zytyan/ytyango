# TODO - Inner HTTP 备份 Zstd 压缩

- [x] Switch `/backupdb` response to tar+zstd stream (e.g. `.tar.zst`) while keeping manifest and db selection.
- [x] Update headers/filenames and PRD usage docs to reflect new format and extraction commands.
- [x] Add/adjust tests for new content-type, filename, and archive extraction checks.
- [x] Run gofmt/go mod tidy if deps added, and go test ./..., noting any failures.
