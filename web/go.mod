// This file exists only to stop Go's ./... pattern (from the repo root
// module) from descending into web/ and picking up stray .go files bundled
// inside npm packages under node_modules (e.g. flatted's Go port). It
// establishes a separate, otherwise-unused module boundary; it has nothing
// to do with the Next.js app itself.
module github.com/dmux/go-quality-gate/web

go 1.21
