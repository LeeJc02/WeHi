SHELL := /bin/bash
GOCACHE ?= $(CURDIR)/.runtime/go-build

.PHONY: start refresh stop smoke verify frontend-install frontend-lint frontend-build go-test local-up local-smoke local-down release-up release-down

start:
	./scripts/compose_up.sh

refresh:
	./scripts/compose_up.sh

stop:
	./scripts/compose_down.sh

smoke:
	./scripts/compose_smoke.sh

verify: go-test
	node --check scripts/runtime_smoke.mjs
	cd frontend && npm run build
	cd frontend && npm run lint

frontend-install:
	cd frontend && npm install

frontend-lint:
	cd frontend && npm run lint

frontend-build:
	cd frontend && npm run build

go-test:
	cd backend && GOCACHE="$(GOCACHE)" go test ./cmd/... ./internal/... ./pkg/... ./services/...

local-up:
	./scripts/compose_up.sh

local-smoke:
	./scripts/compose_smoke.sh

local-down:
	./scripts/compose_down.sh

release-up:
	./scripts/release_up.sh

release-down:
	./scripts/release_down.sh
