// Package version defines the contract for how standalone execution agent modules declare
// their semver for static binary publishing and how they expose it on the CLI.
// The agents core does not import any specific agent implementation.
//
// Semver source (per agent module, independent versions): each agent under code/agents/<slug> has
// its own release line — e.g. linux 0.1.4 and aws 0.1.0 may differ. At that module root, add
// cmd/version.go in package main alongside cmd/main.go with exactly:
//
//	var Version = "x.y.z"
//
// Release tooling (e.g. control plane publish_static_binaries.sh) reads that file per slug
// (see VersionGoFileRelativePath).
//
// CLI: call ExitIfVersionOnly(Version) at the start of main() before flag.Parse() so every agent
// supports -version and --version the same way without duplicating flag handling.
package version

// VersionGoFileRelativePath is relative to the agent module root (directory containing go.mod).
const VersionGoFileRelativePath = "cmd/version.go"

// VersionVarName is the variable identifier in package main within that file.
const VersionVarName = "Version"
