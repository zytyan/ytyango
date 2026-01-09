# Todo: gemini sysprompt format

- [ ] Update gemini sysprompt template to use replacer variables
  - [ ] Skipped: keep %s placeholders to match fmt.Sprintf usage
- [x] Wire gemini_ai.go to use embedded template + fmt.Sprintf
- [x] Update or add tests as needed for new template behavior
- [x] Run gofmt and go test ./... (note any failures)
  - [ ] go test ./... fails: http/backend listens on 127.0.0.1:9892 (address already in use)
