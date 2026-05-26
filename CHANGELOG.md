# Changelog — agent-core

All notable changes to **fluid-pub/agent-core** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Downstream agents should note **agent-core** upgrades in their own changelogs.

## [Unreleased]

### Security

- Repository security aligned with **probe-core**: `SECURITY.md`, Dependabot assignees and grouped GitHub Actions updates, private vulnerability reporting and push protection enabled on GitHub.
- **execution**: validate `fluid_log_path` before `os.Stat` / `os.ReadFile` (CodeQL `go/path-injection`); only absolute paths under `/tmp/fluid/`.
- **execution**: expand `safeFluidLogPath` test coverage (valid/invalid paths, prefix attacks, `startFileLogForwarder` rejection).

## [0.1.0] - 2026-05-26

### Added

- Initial public release: WebSocket execution agent (`execution`, `ws`), enrollment (`enroll`), `skillresult`, `version` CLI contract, `cpcredentials` helpers.
- CI (tests, gofmt) and CodeQL on `develop`.
