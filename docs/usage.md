[![en](https://img.shields.io/badge/lang-en-red.svg)](usage.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](usage.pt-BR.md)

# Go Quality Gate — Usage Guide

A short, practical guide to using quality-gate day to day. For every option, see the [README](../README.md).

## 1. Install the binary

```bash
# macOS (Apple Silicon) — other platforms: see the releases page
curl -fsSL -o quality-gate https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-darwin-arm64
chmod +x quality-gate && sudo mv quality-gate /usr/local/bin/

# or, with Go
go install github.com/dmux/go-quality-gate/cmd/quality-gate@latest

quality-gate --version
```

## 2. Create the configuration

At the root of your repository:

```bash
quality-gate --init
```

It detects your languages and writes a `quality.yml`. Review it and commit it.

## 3. Install the hooks

```bash
quality-gate --install
```

This installs three hooks: `pre-commit` (runs the checks), `commit-msg` (adds the watermark) and `pre-push`.

> Already used quality-gate before v1.3.0? Run `--install` again to get the `commit-msg` hook.

To gate **every** repository on your machine that has a `quality.yml`, instead run:

```bash
quality-gate --install --global
```

## 4. Check the setup

```bash
quality-gate doctor
```

```
✅ binary on PATH: /usr/local/bin/quality-gate
✅ pre-commit hook: .git/hooks/pre-commit
✅ commit-msg hook: .git/hooks/commit-msg
✅ pre-push hook: .git/hooks/pre-push
✅ quality.yml
```

## 5. Commit as usual

```bash
git add .
git commit -m "feat: add login"
```

What happens behind the scenes:

1. **`pre-commit`** runs the checks from `quality.yml`.
   - A failing check **blocks** the commit. Fix it (or run `quality-gate --fix pre-commit`) and try again.
   - If every check passes, quality-gate stores a hash of the staged content and of `quality.yml`.
2. **You write the message** (`-m` or the editor).
3. **`commit-msg`** confirms nothing changed since the checks ran and appends the watermark:
   ```
   feat: add login

   Quality-Gate: v1.3.0; tree=db228f6d…; config=sha256:4feac6c2…; checks=3/3
   ```
   You will see `🔏 Commit watermarked by quality-gate.`
4. **The commit is created** with the watermark.

Check it:

```bash
git log -1 --format='%(trailers)'
quality-gate verify --range HEAD
```

## 6. Special situations

| Situation | What happens |
|---|---|
| `git commit --no-verify` | No hook runs; the commit has **no** watermark and CI rejects it (`missing`). |
| `QG_SKIP="prod hotfix" git commit -m …` | Checks are skipped, but the commit gets `Quality-Gate-Skipped: prod hotfix`. CI accepts it only with `--policy allow-skip`. |
| `git commit --amend` | Hooks run again and the watermark is replaced. With `--no-verify`, the old watermark no longer matches (`tree-mismatch`). |
| Rebase / cherry-pick that changes content | The old watermark no longer matches (`tree-mismatch`); recommit with hooks enabled. |
| Staged files changed between the checks and the message | No watermark, with a warning. Commit again. |
| Empty message (editor closed) | Git aborts as usual; nothing is added. |
| Commit made by an AI agent through MCP | `run_quality_checks` records the watermark too. |

The `commit-msg` hook never blocks a commit; it only warns. Enforcement happens in CI.

## 7. Fixes and JSON output

```bash
quality-gate --fix pre-commit            # run the fix commands
quality-gate pre-commit                  # run the checks by hand
quality-gate --output=json pre-commit    # machine-readable output
```

## 8. Enforce it in CI

Add the Action to your repository (`.github/workflows/quality-gate.yml`):

```yaml
name: Quality Gate
on: pull_request
jobs:
  quality-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: dmux/go-quality-gate@main
        with:
          policy: strict        # or allow-skip
          run-checks: "true"    # re-run the checks in CI
```

Then, in **Settings → Branches (or Rules)**, mark the `quality-gate` job as a **required status check** for `main`. Pull requests with commits that skipped the gate can no longer be merged.

Outside GitHub, run the same command in any CI:

```bash
quality-gate verify --range origin/main..HEAD --policy strict --output json
```

| Status | Meaning |
|---|---|
| `attested` | Passed the gates with this exact content ✅ |
| `skipped` | Bypassed with `QG_SKIP` (accepted only with `allow-skip`) |
| `missing` | No watermark (hooks not installed or `--no-verify`) |
| `tree-mismatch` | Content changed after the gates ran |
| `config-mismatch` | Watermark from a different `quality.yml` |
| `malformed` | Unreadable watermark |

## 9. FAQ

**What is a trailer?** A `Key: value` line at the end of a commit message, after a blank line — the same convention as `Signed-off-by` and `Co-authored-by`. It travels with the commit and git can read it natively (`git interpret-trailers`, `%(trailers)`).

**Can the watermark be forged?** Yes, someone could type it by hand. It stops casual bypasses and leaves an audit trail; the required CI check, which re-runs the gates, is what makes the gate mandatory.

**Squash merge?** It creates a new commit without a watermark. Verify the pull request commits, not the merge commit — the Action does this by default.
