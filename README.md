<!-- Generated from private documentation source. Do not edit directly. Source SHA256: b6967083c379daafaedb0c98745da29f86c61950cf31a92c63f59c99e3d85035 -->

# GDLint

`gdlint` is a project-local GDScript hygiene checker for Godot projects.

It is intentionally not an architecture checker. It does not know about workspace layouts, shared source folders, directory naming conventions, import layering, or autoload ownership rules. Those concerns belong in project-specific architecture tools.

## Install

```bash
go install github.com/aviorstudio/gdlint@latest
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

## Development

Install the [Komizo release](https://github.com/nicodes/komizo#install) matching `komizo.lock`.

```sh
komizo check        # Complete project checks
komizo run build    # Build the declared distributable artifacts
komizo describe     # Inspect tools and focused stages
```
