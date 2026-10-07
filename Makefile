.DEFAULT_GOAL := help
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.NOTPARALLEL:
VERSION ?= dev
export VERSION
.PHONY: help install lint test build artifact-smoke check dev stop clean
help:
	@echo 'make install: pinned tools; make check: source, CLI and all release archives'
install:
	mise trust .mise.toml
	mise install
	mise exec -- go mod download
lint:
	mise exec -- bash scripts/lint.sh
build:
	mise exec -- bash scripts/build-release.sh
artifact-smoke: build
	mise exec -- python3 scripts/check_archives.py
# The fixture exercises the archived host binary, not a separately built copy.
test: artifact-smoke
	mise exec -- go test -race -count=1 ./...
	mise exec -- python3 scripts/test-cli.py
check: lint test
dev stop:
	@echo '$@: unsupported: one-shot CLI has no development service'
clean:
	mise exec -- python3 -c 'import shutil; [shutil.rmtree(p, ignore_errors=True) for p in ("bin", "dist", ".artifacts")]'
