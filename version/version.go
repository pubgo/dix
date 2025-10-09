package version

import (
	_ "embed"
)

//go:embed .version
var version string

func ReleaseVersion() string { return version }

func ReleaseDate() string { return "2025-10-08" }
