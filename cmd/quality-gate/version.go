package main

import "fmt"

// Version information for the quality-gate tool
// These variables are set during build time via -ldflags
var (
	// Version is the current version of the quality-gate tool
	Version = "1.2.0-dev"

	// BuildDate is set during build time
	BuildDate = "development"

	// GitCommit is set during build time
	GitCommit = "development"
)

// VersionInfo returns formatted version information
func VersionInfo() string {
	return fmt.Sprintf("quality-gate version %s", Version)
}

// DetailedVersionInfo returns detailed version information including build details
func DetailedVersionInfo() string {
	return fmt.Sprintf(`quality-gate version %s
Build Date: %s
Git Commit: %s`, Version, BuildDate, GitCommit)
}
