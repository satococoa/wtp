//go:build tools
// +build tools

// Package tools imports development tools for dependency tracking.
// This file is not compiled as part of the main application.
package tools

import (
	// Linting tool
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"

	// Import management tool
	_ "golang.org/x/tools/cmd/goimports"

	// Test coverage reporting
	_ "golang.org/x/tools/cmd/cover"

	// Release tool
	_ "github.com/goreleaser/goreleaser"
)
