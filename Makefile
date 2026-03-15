GO ?= go
NPM ?= npm

.PHONY: backend-run backend-test frontend-install frontend-dev frontend-build frontend-lint frontend-smoke smoke

backend-run:
	cd backend && $(GO) run ./cmd/api

backend-test:
	cd backend && $(GO) test ./...

frontend-install:
	cd frontend && $(NPM) install

frontend-dev:
	cd frontend && $(NPM) run dev

frontend-build:
	cd frontend && $(NPM) run build

frontend-lint:
	cd frontend && $(NPM) run lint

frontend-smoke:
	cd frontend && $(NPM) run smoke -- http://127.0.0.1:8081

smoke: backend-test frontend-lint frontend-build
