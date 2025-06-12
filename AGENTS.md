# Repo Agent Instructions

This repository is written in Go and uses a single module defined in `go.mod`.

* Run `go mod tidy` when dependencies change to keep module files up to date.
* Format Go code with `gofmt -w`.
* Run `go test ./...` before creating a pull request to test all packages. If tests fail due to missing dependencies or external services, mention this in the PR body.
* Keep commit messages short and descriptive in English.
