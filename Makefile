.PHONY: fmt test test-race build web-install web-generate web-check compose-up

fmt:
	gofmt -w apps internal

test:
	go test ./...

test-race:
	go test -race ./...

build:
	go build -o bin/codex-tempo-agent ./apps/agent
	go build -o bin/codex-tempo ./apps/cli
	go build -o bin/codex-tempo-server ./apps/server

web-install:
	pnpm install

web-generate:
	pnpm generate:api

web-check:
	pnpm lint
	pnpm typecheck
	pnpm build

compose-up:
	docker compose -f deploy/compose/docker-compose.yml up --build
