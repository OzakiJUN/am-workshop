# CLAUDE.md — am-workshop

Index repository of the AM creative workshop. The app itself is
https://github.com/OzakiJUN/AM: read its CLAUDE.md (how Weiqi wants to work:
casual Chinese, plan first, docs are the source of truth) and its
docs/design.md §13 (the workshop design). AGENTS.md says the same for other agents.

- `items/<id>.json`: one work each (format: `Item` in `tools/workshop/format.go`).
- Package rules live in `tools/workshop/format.go` and FORMAT.md; the AM
  client checks the same rules in `client/src/lib/workshopPackage.ts`. Change
  all three together.
- `check.yml` runs the checker built from the base branch (a PR can't change
  how it is checked); non-maintainers may only touch their own items.
- `publish.yml` builds `index.json`, signs it with Ed25519 (secret
  `WORKSHOP_SIGNING_KEY` = base64 seed from `am-server workshop-keygen`) and
  force-pushes it to the `index` branch. The public key is built into the client.
- Checks: `go vet ./... && go test ./...`.
