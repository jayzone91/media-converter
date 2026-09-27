set windows-shell := ["powershell.exe", "-NoLogo", "-NoProfile", "-Command"]

default:
    @just --list

# Frontend-Abhängigkeiten installieren.
install:
    bun install

# templ Go-Code generieren.
generate:
    templ generate

# TypeScript vollständig prüfen.
typecheck:
    bun run typecheck

# Frontend für Entwicklung einmalig bauen.
frontend-dev:
    bun run build:dev

# Frontend minifiziert für Produktion bauen.
frontend:
    bun run build

# Go formatieren.
fmt:
    templ fmt .
    go fmt ./...

# Go Tests.
test: generate
    go test ./...

# Produktions-Build.
build: generate frontend
    go test ./...
    go build ./cmd/media-converter

# Entwicklungs-Build vorbereiten und Server starten.
dev: generate frontend-dev
    go run ./cmd/media-converter

# TypeScript beobachten.
watch-js:
    bun run watch:js

# SCSS beobachten.
watch-css:
    bun run watch:css
