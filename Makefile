SHELL := /bin/bash
GOCACHE ?= $(CURDIR)/.runtime/go-build

.PHONY: start refresh stop smoke frontend-install frontend-lint frontend-build go-test local-up local-smoke local-down

start:
	./scripts/compose_up.sh

refresh:
	./scripts/compose_up.sh

stop:
	./scripts/compose_down.sh

smoke:
	./scripts/compose_smoke.sh

frontend-install:
	cd frontend && npm install

frontend-lint:
	cd frontend && npm run lint

frontend-build:
	cd frontend && npm run build

go-test:
	GOCACHE="$(GOCACHE)" go test ./cmd/... ./internal/... ./pkg/... ./services/...

local-up:
	./scripts/compose_up.sh

local-smoke:
	./scripts/compose_smoke.sh

local-down:
	./scripts/compose_down.sh
