package main

import (
	"runtime/debug"
)

var readBuildInfo = debug.ReadBuildInfo

func initVersion() {
	if version != defaultVersion {
		return
	}

	info, ok := readBuildInfo()
	if !ok || info == nil {
		return
	}

	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return
	}

	version = info.Main.Version
}
