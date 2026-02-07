package version

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// Version is the service current released version.
// This value can be overridden at build time using ldflags:
//
//	go build -ldflags "-X github.com/hrygo/divinesense/internal/version.Version=v0.95.0"
//
// Semantic versioning: https://semver.org/
var Version = "0.0.0-dev"

// DevVersion is the service current development version.
var DevVersion = Version

// GitCommit is the git commit hash at build time.
// Set via ldflags: -X github.com/hrygo/divinesense/internal/version.GitCommit=$(git rev-parse HEAD)
var GitCommit = "unknown"

// GitBranch is the git branch at build time.
// Set via ldflags: -X github.com/hrygo/divinesense/internal/version.GitBranch=$(git rev-parse --abbrev-ref HEAD)
var GitBranch = "unknown"

// BuildTime is the build timestamp in RFC3339 format.
// Set via ldflags: -X github.com/hrygo/divinesense/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)
var BuildTime = "unknown"

func GetCurrentVersion(mode string) string {
	if mode == "dev" || mode == "demo" {
		return DevVersion
	}
	return Version
}

// GetMinorVersion extracts the minor version (e.g., "0.25") from a full version string (e.g., "0.25.1").
// Returns the minor version string or empty string if the version format is invalid.
// Version format should be "major.minor.patch" (e.g., "0.25.1").
func GetMinorVersion(version string) string {
	versionList := strings.Split(version, ".")
	if len(versionList) < 2 {
		return ""
	}
	// Return major.minor only (first two components)
	return versionList[0] + "." + versionList[1]
}

// IsVersionGreaterOrEqualThan returns true if version is greater than or equal to target.
func IsVersionGreaterOrEqualThan(version, target string) bool {
	return semver.Compare(fmt.Sprintf("v%s", version), fmt.Sprintf("v%s", target)) > -1
}

// IsVersionGreaterThan returns true if version is greater than target.
func IsVersionGreaterThan(version, target string) bool {
	return semver.Compare(fmt.Sprintf("v%s", version), fmt.Sprintf("v%s", target)) > 0
}

type SortVersion []string

func (s SortVersion) Len() int {
	return len(s)
}

func (s SortVersion) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SortVersion) Less(i, j int) bool {
	v1 := fmt.Sprintf("v%s", s[i])
	v2 := fmt.Sprintf("v%s", s[j])
	return semver.Compare(v1, v2) == -1
}

// String returns the version string with optional commit hash.
func String() string {
	v := Version
	if GitCommit != "" && GitCommit != "unknown" {
		shortCommit := GitCommit
		if len(shortCommit) > 8 {
			shortCommit = shortCommit[:8]
		}
		v = fmt.Sprintf("%s-%s", v, shortCommit)
	}
	return v
}

// StringFull returns the complete version information including build metadata.
func StringFull() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Version=%s", Version))
	if GitCommit != "" && GitCommit != "unknown" {
		shortCommit := GitCommit
		if len(shortCommit) > 8 {
			shortCommit = shortCommit[:8]
		}
		parts = append(parts, fmt.Sprintf("Commit=%s", shortCommit))
	}
	if GitBranch != "" && GitBranch != "unknown" {
		parts = append(parts, fmt.Sprintf("Branch=%s", GitBranch))
	}
	if BuildTime != "" && BuildTime != "unknown" {
		parts = append(parts, fmt.Sprintf("BuildTime=%s", BuildTime))
	}
	return strings.Join(parts, " ")
}
