# AGENTS.md

Guidance for AI coding agents and contributors working on go-quality-gate. Write all code, comments, commits, PRs and docs in English (the Portuguese READMEs and usage guides are the only exception).

## Project

A Go CLI that installs git hooks and runs the quality checks configured in `quality.yml`; it also exposes an MCP server and a small Next.js dashboard in `web/`.

- `cmd/quality-gate`: CLI entrypoint and `version.go`
- `internal/{config,domain,repository,service}`: clean-architecture layers; `repository` holds interfaces, `service` holds logic
- `internal/infra/*`: git, shell, logger, history, spinner, web UI adapters
- `internal/mcp`: MCP server
- `web/`: Next.js dashboard (has its own `AGENTS.md`)
- `docs/`: GitHub Pages site (`index.html`) and usage guides

## Branching and workflow

- **Always start from `develop`** and open PRs against `develop`. Never commit directly to `main` or `develop`.
- `main` only receives `develop` through a `develop` → `main` PR, and only for releases or urgent fixes.
- Branch names: `feat/…`, `fix/…`, `chore/…`, `docs/…`, `ci/…`, `release/x.y.z`.
- Keep PRs small and focused; one concern per PR. PR titles and descriptions are in English, with a short summary and a test plan.
- Commits follow Conventional Commits, as the history does: `feat(service): …`, `fix(doctor): …`, `chore(deps): …`, `docs: …`.
- Do not add AI/Claude attribution lines (`Co-Authored-By`, "Generated with …") to commits, PRs or release notes.
- Never push or merge without being asked. Pushing a `v*` tag publishes a release and cannot be cleanly undone.

## Development practices

- Before opening a PR run: `go build ./... && go vet ./... && gofmt -s -l . && go test -race ./...` (the output of `gofmt -s -l .` must be empty).
- Match the surrounding code: naming, comment density and the existing test style (stub executables on `PATH`, no mocking framework).
- New behavior needs tests; bug fixes need a regression test.
- Config changes must stay backward compatible: new `quality.yml` settings need a safe default, validation in `internal/config/validator.go`, and docs.
- Never execute user-defined commands from read-only checks (`doctor`, `check_environment`).
- User-facing changes update `README.md`, `README.pt-BR.md`, `docs/usage*.md` and `CHANGELOG.md`.
- Keep `quality.yml` examples and the `--init` template in sync with the config schema.

## Web (`web/`)

- Install with `npm ci --legacy-peer-deps` (a pre-existing peer conflict between `vite` and `@types/node` blocks plain `npm install`).
- Verify with `npm run lint`, `npm test` and `npm run build`. **CI does not run these**, so run them locally for any change under `web/`.
- Dependabot groups `/web` updates into one PR and ignores majors of `typescript`, `eslint` and `@types/node`; those need their own reviewed PR.

## CI

- `ci.yml` runs vet, tests with coverage and build on PRs to `main` and `develop`; `build-test.yml` cross-compiles for 5 platforms and builds the Docker image.
- Do not merge with a red or still-pending check. A green check says nothing about `web/` (see above).

## Release process

1. Branch `release/x.y.z` from `develop` (or from `main` for hotfixes) and bump the version in: `Makefile` (`VERSION?=x.y.z`), `cmd/quality-gate/version.go` (`x.y.z-dev`; release binaries take the version from the tag), the version examples in `README.md`, `README.pt-BR.md`, `docs/usage.md`, `docs/usage.pt-BR.md`, and the badge and example in `docs/index.html` (GitHub Pages).
2. Add a section to `CHANGELOG.md` (Keep a Changelog style: Added / Changed / Fixed / Dependencies), and thank external contributors briefly by handle.
3. Run the pre-PR checks and `make build && ./quality-gate -v`, open a `chore: release x.y.z` PR to `main`, wait for CI, merge with a merge commit.
4. Tag the merged `main` commit with an annotated `vx.y.z` tag and push it. `release.yml` builds the binaries and checksums, creates the GitHub release and pushes the Docker image to ghcr.io.
5. Replace the generic release notes with a short highlights list and thanks (`gh release edit vx.y.z --notes-file …`).
6. Fast-forward `develop` to `main` (`git merge --ff-only origin/main`) so both branches match.

Use semver: new features bump the minor version, fixes only bump the patch.
