package version

import (
	"strings"
	"testing"
)

func TestGetVersion_DefaultValues(t *testing.T) {
	// Reset to defaults for testing
	origVersion := Version
	origGitCommit := GitCommit
	origGitTag := GitTag
	defer func() {
		Version = origVersion
		GitCommit = origGitCommit
		GitTag = origGitTag
	}()

	Version = "dev"
	GitCommit = "unknown"
	GitTag = "unknown"

	got := GetVersion()
	if got != "dev" {
		t.Errorf("GetVersion() with defaults = %v, want %v", got, "dev")
	}
}

func TestGetVersion_WithLdflags(t *testing.T) {
	// Simulate ldflags setting the version
	origVersion := Version
	defer func() { Version = origVersion }()

	Version = "v1.2.3"

	got := GetVersion()
	if got != "v1.2.3" {
		t.Errorf("GetVersion() with ldflags = %v, want %v", got, "v1.2.3")
	}
}

func TestGetVersion_WithGitInfo(t *testing.T) {
	// Simulate git tag and commit
	origVersion := Version
	origGitCommit := GitCommit
	origGitTag := GitTag
	origGitDirty := GitDirty
	defer func() {
		Version = origVersion
		GitCommit = origGitCommit
		GitTag = origGitTag
		GitDirty = origGitDirty
	}()

	Version = "dev"
	GitTag = "v1.2.3"
	GitCommit = "abc1234567"
	GitDirty = ""

	got := GetVersion()
	// Should construct from git info: tag-commit
	if !strings.HasPrefix(got, "v1.2.3") {
		t.Errorf("GetVersion() with git info = %v, want prefix %v", got, "v1.2.3")
	}
	if !strings.Contains(got, "abc1234") {
		t.Errorf("GetVersion() with git info = %v, want to contain %v", got, "abc1234")
	}
}

func TestGetVersion_WithDirtyFlag(t *testing.T) {
	origVersion := Version
	origGitCommit := GitCommit
	origGitTag := GitTag
	origGitDirty := GitDirty
	defer func() {
		Version = origVersion
		GitCommit = origGitCommit
		GitTag = origGitTag
		GitDirty = origGitDirty
	}()

	Version = "dev"
	GitTag = "v1.2.3"
	GitCommit = "abc1234"
	GitDirty = "dirty"

	got := GetVersion()
	if !strings.HasSuffix(got, "-dirty") {
		t.Errorf("GetVersion() with dirty flag = %v, want suffix %v", got, "-dirty")
	}
}

func TestGetFullVersion(t *testing.T) {
	origVersion := Version
	origGitCommit := GitCommit
	defer func() {
		Version = origVersion
		GitCommit = origGitCommit
	}()

	Version = "v1.2.3"
	GitCommit = "abc1234"

	got := GetFullVersion()
	if !strings.Contains(got, "v1.2.3") {
		t.Errorf("GetFullVersion() = %v, want to contain %v", got, "v1.2.3")
	}
	if !strings.Contains(got, "abc1234") {
		t.Errorf("GetFullVersion() = %v, want to contain %v", got, "abc1234")
	}
}

func TestGetBuildInfo(t *testing.T) {
	origVersion := Version
	origGitCommit := GitCommit
	origGitTag := GitTag
	origBuildTime := BuildTime
	origGitDirty := GitDirty
	defer func() {
		Version = origVersion
		GitCommit = origGitCommit
		GitTag = origGitTag
		BuildTime = origBuildTime
		GitDirty = origGitDirty
	}()

	Version = "v1.2.3"
	GitCommit = "abc1234"
	GitTag = "v1.2.3"
	BuildTime = "2025-01-01T00:00:00Z"
	GitDirty = ""

	info := GetBuildInfo()

	// Check all keys exist
	expectedKeys := []string{"version", "gitCommit", "gitTag", "buildTime", "gitDirty"}
	for _, key := range expectedKeys {
		if _, ok := info[key]; !ok {
			t.Errorf("GetBuildInfo() missing key %v", key)
		}
	}

	// Check values
	if info["version"] != "v1.2.3" {
		t.Errorf("GetBuildInfo()[version] = %v, want %v", info["version"], "v1.2.3")
	}
	if info["gitCommit"] != "abc1234" {
		t.Errorf("GetBuildInfo()[gitCommit] = %v, want %v", info["gitCommit"], "abc1234")
	}
}
