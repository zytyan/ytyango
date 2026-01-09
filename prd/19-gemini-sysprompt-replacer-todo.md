# Todo: gemini sysprompt replacer

- [x] Update gemini_sysprompt.txt to use %VAR% placeholders
  - [x] Use %DATETIME_TZ% to preserve timezone offset format
- [x] Update gemini_ai.go to build sys prompt via helpers/replacer
- [x] Update or add tests to cover replacer-based formatting
- [x] Run gofmt and go test ./... (note any failures)
