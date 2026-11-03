<!-- Generated from private documentation source. Do not edit directly. Source SHA256: bca842ddc8f3812eccc4776e12b6e1803bb60cf7aa63ce9f259a2428f0e004a3 -->

# gdlint

`gdlint` is a project-local GDScript hygiene checker for Godot projects.

It is intentionally not an architecture checker. It does not know about workspace layouts, shared source folders, directory naming conventions, import layering, or autoload ownership rules. Those concerns belong in project-specific architecture tools.

## Install

Linux and macOS (x86_64 or arm64), using prebuilt binaries:

With curl:

```sh
curl -fsSL https://gdlint.dev/install | sh
```

Or with mise:

```sh
mise use --global github:aviorstudio/gdlint@latest
```

The shell installer verifies the release checksum and installs to `~/.local/bin`. Add that directory to your PATH if necessary. Mise manages its own installation directory and PATH through its shell activation.

## Usage

Run from a Godot project root:

```bash
cd path/to/godot_project
gdlint
```

The current directory must contain `project.godot`.

`gdlint` loads `./gdlint.json` when present. If no config exists, it uses mild generic defaults.

## Rule Ownership

`gdlint` owns code-hygiene checks inside valid Godot project files:

- unused functions, signals, constants, enums, and files
- print/debug statements
- comments outside allowed prefixes
- unnecessary `pass` statements
- orphaned `.gd.uid` files
- indentation
- dynamic `has_method()` usage
- explicit `Variant` usage
- missing public return type annotations
- empty functions and single-use functions as optional warnings

`gdlint` does not own architecture checks:

- directory naming and required sibling files
- `_util`, `_state`, `_service`, `_autoload`, route, layout, or component semantics
- import direction rules
- autoload registration and manual autoload loading
- UID allowed/forbidden policy by directory
- workspace-specific source relationships

Use a project-specific architecture checker for those rules.

## Config

Create a config in the current Godot project:

```bash
gdlint init
```

This writes `./gdlint.json` and fails if the file already exists.

Config sections:

- `errors`: blocking checks
- `warnings`: non-blocking checks shown with `-warn`
- `settings.allowed_comment_prefixes`: comment exceptions
- `settings.ignore_patterns`: files skipped by all rules
- `settings.unused_ignore_patterns`: files skipped only by unused-code rules
- `settings.variant_ignore_patterns`: files skipped only by explicit `Variant` checks

## Flags

```bash
gdlint
gdlint -warn
gdlint -debt
gdlint -benchmark
gdlint init
gdlint version
gdlint -fix
```

Use `-fix` carefully. It can remove code and files when static analysis believes they are unused. The fixer refuses symlinked files and checks removal ranges before writing. Rules without an automatic fix remain reported after the existing ten-iteration limit.

## Build from source

The CLI is implemented in Rust. With Rust 1.94.0 installed, run:

```sh
cargo build --release --locked
./target/release/gdlint version
```

## Contributor verification

Tool versions and native tasks live in `mise.toml`. Prepare the pinned helper source and run the complete gate:

```sh
git clone https://github.com/nicodes/tools.git .ci-tools
git -C .ci-tools checkout --detach 091101094fedf5ce2e67c339c1c9c842d6a662dc
mise install
mise run check
```

Native build and test commands retain source and output checks, with receipts in `.artifacts/verification`. Existing product release rules and artifacts remain owned by this repository. End users install the product’s published distribution.

## Contributor task interface

Contributor tool versions and task entry points live in the native Mise configuration. Use `mise run lint` for source checks, `mise run test` for the applicable behavioral tests, and `mise run check` for the complete supported gate. `mise run clean` removes declared generated outputs while preserving source, credentials and retained data. This package has no separate Mise development service; use its native entry points directly.

## Machine results and configuration

Use `gdlint --json` for structured findings, `gdlint --json -fix` for a verified fix result, and `gdlint --json version` for build identity. Machine stdout contains one JSON document without terminal control bytes. Diagnostics use stderr. Errors in the project and unresolved fixes exit 1; invalid command syntax exits 2; help/version succeed.

Configuration defaults to `gdlint.json` in the Godot project root. `--config FILE` selects an explicit file; an explicitly missing file fails. Project settings stay with the project rather than user-global configuration.
