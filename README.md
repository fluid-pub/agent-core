# fluid-pub/agent-core

Shared Go library for Fluid **execution agents**: WebSocket control plane client, enrollment (`POST /api/v1/enrollment/enroll`), skill execution helpers, and durable credentials patterns.

Downstream agents (`fluid-pub/agent-*`) submodule this repo at `core/` and use `replace fluid/agents/core => ./core` in `go.mod`.

## Layout

| Package | Role |
|---------|------|
| `execution` | WebSocket agent loop, `skill_invoke` / `skill_result`, log events |
| `ws` | Control plane WebSocket client |
| `enroll` | First-boot enrollment HTTP client |
| `skillresult` | Standard skill success/failure payloads |
| `version` | `cmd/version.go` contract and `-version` CLI helper |
| `cpcredentials` | Optional control-plane–issued credentials helpers |

## Consumers

Import as `fluid/agents/core` (module path in `go.mod`).

## Releases

Workloads pin `core_ref: develop` in CI or the same semver tag as the agent on release. Changelog: [CHANGELOG.md](CHANGELOG.md).

## Local development (Fluid monorepo)

Source of truth during development: `code/agents/core/` in the Fluid workspace. Publish updates here before tagging downstream agents that depend on new APIs.
