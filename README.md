<!-- Generated from private documentation source. Do not edit directly. Source SHA256: 1b696ec8337ddabe32c7831bee500c4c381d0740a623eceeff234f0ea8118bc3 -->

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

The CLI is implemented in Rust. Prepare the pinned toolchain and build its verified native executable:

```sh
make setup
make build
.artifacts/build/gdlint version
```

## Contributor verification

`mise.toml` pins contributor SDKs. The generated root Makefile selects an immutable revision of the shared cookbook and binds it to `cookbook.toml`.

```sh
make help
make setup
make build
make test
```

`make help` works offline before setup. `make setup` explicitly prepares the pinned SDKs, locked dependencies and declared fixtures. `make build` compiles the executable, packages the native archive and verifies its source and output identity. `make test` runs lint, deterministic native unit tests, security checks and the declared integration journeys against those verified artifacts. Run build again after changing source; tests do not install tools or build missing shipping artifacts.

Use `make fmt` to apply formatting, `make test:lint`, `make test:unit`, `make test:security` or `make test:integration` to select a group, and `make clean` to remove the declared generated outputs. Build and test evidence is recorded in `.artifacts/verification`.

The verified executable is `.artifacts/build/gdlint` (`gdlint.exe` on Windows). CI retains Linux, macOS and Windows builds for both x86_64 and arm64. Published Linux binaries use musl, and Windows binaries include the C runtime. Release archives retain their existing contents, checksums and source-workflow build provenance. Integration journeys exercise native CLI behavior, archived binaries and the installer.

## Machine results and configuration

Use `gdlint --json` for structured findings, `gdlint --json -fix` for a verified fix result, and `gdlint --json version` for build identity. Machine stdout contains one JSON document without terminal control bytes. Diagnostics use stderr. Errors in the project and unresolved fixes exit 1; invalid command syntax exits 2; help/version succeed.

Configuration defaults to `gdlint.json` in the Godot project root. `--config FILE` selects an explicit file; an explicitly missing file fails. Project settings stay with the project rather than user-global configuration.
