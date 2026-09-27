set windows-shell := ["powershell.exe", "-NoLogo", "-NoProfile", "-Command"]

default:
    @just --list

# Frontend-Abhängigkeiten installieren.
install:
    bun install

# templ Go-Code generieren.
generate:
    templ generate

# Maximale Dateigröße von 500 LOC prüfen.
check-loc:
    go run ./cmd/checkloc

# TypeScript vollständig prüfen.
typecheck:
    bun run typecheck

# Frontend für Entwicklung einmalig bauen.
frontend-dev:
    bun run build:dev

# Frontend minifiziert für Produktion bauen.
frontend:
    bun run build

# Go und templ formatieren.
fmt:
    templ fmt .
    go fmt ./...

# Qualitätsprüfungen ohne Build.
check: generate
    go run ./cmd/checkloc
    bun run typecheck
    go test ./...

# Go Tests.
test: generate
    go run ./cmd/checkloc
    go test ./...

# Produktions-Build.
build: generate frontend
    go run ./cmd/checkloc
    go test ./...
    go build ./cmd/media-converter

# Entwicklungs-Build vorbereiten und Server starten.
dev: generate frontend-dev
    go run ./cmd/checkloc
    go run ./cmd/media-converter

# TypeScript beobachten.
watch-js:
    bun run watch:js

# SCSS beobachten.
watch-css:
    bun run watch:css
