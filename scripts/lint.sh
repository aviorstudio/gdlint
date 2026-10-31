#!/usr/bin/env bash
set -euo pipefail
cargo fmt --check
cargo clippy --locked --all-targets -- -D warnings
shellcheck scripts/*.sh
actionlint
