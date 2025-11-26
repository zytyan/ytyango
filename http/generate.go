package http

//go:generate go run github.com/ogen-go/ogen/cmd/ogen@latest --target backend/botapi --package botapi --clean botapi.yaml

//go:generate npx openapi-typescript botapi.yaml -o frontend/src/api/schema.ts
