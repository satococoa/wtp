package main

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitVersionUsesBuildInfoWhenDev(t *testing.T) {
	prevVersion := version
	prevReader := readBuildInfo
	t.Cleanup(func() {
		version = prevVersion
		readBuildInfo = prevReader
	})

	version = defaultVersion
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/satococoa/wtp/v2",
				Version: "v2.3.4",
			},
		}, true
	}

	initVersion()

	assert.Equal(t, "v2.3.4", version)
}

func TestInitVersionIgnoresDevelVersion(t *testing.T) {
	prevVersion := version
	prevReader := readBuildInfo
	t.Cleanup(func() {
		version = prevVersion
		readBuildInfo = prevReader
	})

	version = defaultVersion
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/satococoa/wtp/v2",
				Version: "(devel)",
			},
		}, true
	}

	initVersion()

	assert.Equal(t, defaultVersion, version)
}

func TestInitVersionRespectsPresetVersion(t *testing.T) {
	prevVersion := version
	prevReader := readBuildInfo
	t.Cleanup(func() {
		version = prevVersion
		readBuildInfo = prevReader
	})

	version = "custom"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/satococoa/wtp/v2",
				Version: "v2.3.4",
			},
		}, true
	}

	initVersion()

	assert.Equal(t, "custom", version)
}
