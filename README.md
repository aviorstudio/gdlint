# GDLint

`gdlint` is a project-local GDScript hygiene checker for Godot projects.

It is intentionally not an architecture checker. It does not know about workspace layouts, shared source folders, directory naming conventions, import layering, or autoload ownership rules. Those concerns belong in project-specific architecture tools.

## Install

```bash
go install github.com/aviorstudio/gdlint@latest
```

For local development in this repository:

```bash
go build -o bin/gdlint
```

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

Use `-fix` carefully. It can remove code and files when static analysis believes they are unused.

## Release

Releases are created manually from GitHub Actions using a `major`, `minor`, or `patch` dropdown. The workflow calculates the next `vX.Y.Z` tag from existing release tags. If no release tag exists, the first release is `v0.0.1`.

The release workflow:

- runs `go test ./...`
- builds Linux, macOS, and Windows binaries for `amd64` and `arm64`
- injects version, commit, and build date into `gdlint version`
- packages archives with this README
- generates `checksums.txt`
- creates and pushes the git tag
- publishes a GitHub release
