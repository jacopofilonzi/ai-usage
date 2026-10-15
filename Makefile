# Recipes must work in both sh and cmd.exe: only `cd dir && cmd`, no inline VAR=x, no rm/cp/mkdir -p.
# Environment for the go tool is set with target-specific exports instead.

# Packages without Wails (which needs cgo on Linux/macOS): vetted for every OS from any host.
CROSS_PKGS := ./internal/config/... ./internal/provider/... ./internal/wakelock/... ./internal/inhibit/... \
	./internal/autostart/... ./internal/logging/... ./internal/poller/... ./internal/trayicon/... \
	./internal/i18n/... ./internal/platform/...
# Packages that import Wails: vetted for Linux/macOS with its cgo-free server build tag, which at
# least type-checks the non-Windows files against the Wails API.
WAILS_PKGS := . ./internal/app/... ./internal/tray/...

.DEFAULT_GOAL := help
.PHONY: help setup dev build bindings frontend frontend-check test fmt fmt-check vet vet-linux vet-darwin check

help:
	@echo make setup            install frontend dependencies (once, and after package.json changes)
	@echo make dev              run the app with hot reload (wails3 dev)
	@echo make build            production build into bin/
	@echo make bindings         regenerate the TypeScript bindings in frontend/bindings
	@echo make test             Go unit tests
	@echo make fmt              format the Go code
	@echo make check            gofmt + vet (windows, linux, darwin) + tests + svelte-check: must pass before done

setup:
	cd frontend && npm install

dev:
	wails3 dev -config ./build/config.yml -port 9245

build:
	wails3 task build

bindings:
	wails3 generate bindings -clean=true -ts -i

# The Go build embeds frontend/dist, so it must exist before vet/test.
frontend:
	cd frontend && npm run build

frontend-check:
	cd frontend && npm run check

test:
	go test ./...

fmt:
	gofmt -w main.go internal

fmt-check:
	$(if $(strip $(shell gofmt -l main.go internal)),$(error gofmt needed on: $(shell gofmt -l main.go internal)))
	@echo gofmt ok

vet:
	go vet ./...

vet-linux: export GOOS := linux
vet-linux: export CGO_ENABLED := 0
vet-linux:
	go vet $(CROSS_PKGS)
	go vet -tags server $(WAILS_PKGS)

vet-darwin: export GOOS := darwin
vet-darwin: export CGO_ENABLED := 0
vet-darwin:
	go vet $(CROSS_PKGS)
	go vet -tags server $(WAILS_PKGS)

check: fmt-check bindings frontend-check frontend vet vet-linux vet-darwin test
