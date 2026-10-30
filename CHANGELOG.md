# Changelog

All notable changes to this project are documented in this file.

## [1.4.2] - 2026-10-29

### Changed

- **Breaking:** the commit watermark is now disabled by default. Repositories that rely on it (for example with `quality-gate verify` in CI) must add `settings.watermark: true` to `quality.yml`. While disabled, no attestation is recorded, `commit-msg` leaves the message untouched, `QG_SKIP` adds no trailer, and the GitHub Action skips the verify step.
- `quality-gate --init` now documents the `settings.watermark` switch in the generated `quality.yml`.

### Documentation

- Added `AGENTS.md` (and a `CLAUDE.md` that imports it) with the branching workflow, project practices and release process.

## [1.4.1] - 2026-10-29

### Added

- `settings.tools_policy` in `quality.yml`: `install` (default) auto-installs missing tools, `recommend` only prints the install command.
- Runtime detection for Go, Python, Node, Rust, PHP and Java, with install hints for macOS, Windows and Linux (`apt-get`, `dnf`, `yum`, `apk`, `pacman`).
- `quality-gate doctor` now checks configured tools and language runtimes, and reports warnings and skipped checks separately from failures.
- `check_environment` MCP tool, sharing the same checks as `doctor`.
- `--init` asks for the project language when none is detected in an interactive terminal.
- `--update` flag to upgrade quality-gate to the latest version.

### Fixed

- `doctor` no longer exits with an error on a fresh clone when a tool will be installed automatically.
- `doctor` no longer reports `npx`, `python -m`, `go run` and similar launcher commands as verified.
- The Python runtime is found as `python` or `py` on Windows.
- The language detector no longer skips its own root directory when it is dot-prefixed.
- The tool-install cache is keyed on the tools policy, and recommendations are no longer silenced by it.

### Dependencies and CI

- Web dashboard dependencies updated, including `next` 16.4.3, `react` 19.3.1, `vite` 8.3, `vitest` 5.0.5 and `shadcn` 4.23 (fixes the `zod` advisory); security advisories for `next`, `proxy-addr`, `@modelcontextprotocol/sdk` and others are resolved.
- Dependabot groups `/web` updates into a single pull request and ignores major bumps of `typescript`, `eslint` and `@types/node`.
- CI now runs on pull requests targeting `develop`.

### Thanks

Thank you to [@mateusememe](https://github.com/mateusememe) for the environment-aware `doctor`, `tools_policy` and `check_environment` work, and to [@alini-x](https://github.com/alini-x) for `--update`.

## [1.4.0]

See the [v1.4.0 release](https://github.com/dmux/go-quality-gate/releases/tag/v1.4.0).
