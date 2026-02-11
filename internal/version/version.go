// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package version

import (
	"runtime"

	"github.com/MakeNowJust/heredoc"
)

var (
	Version   = "dev"
	Commit    = "unset"
	BuildTime = "unknown"
)

func String() string {
	return heredoc.Docf(`
		Unikraft CLI
		  version:    %s
		  commit:     %s
		  platform:   %s/%s
		  build time: %s
		  go version: %s
		  docs:       https://unikraft.com/docs
		  issues:     https://github.com/unikraft/cli/issues`,
		Version,
		Commit,
		runtime.GOOS,
		runtime.GOARCH,
		BuildTime,
		runtime.Version(),
	)
}

// UserAgent returns the user agent string for the Unikraft CLI.
func UserAgent() string {
	return heredoc.Docf(
		"unikraft-cli/%s (%s) %s/%s",
		Version,
		Commit,
		runtime.GOOS,
		runtime.GOARCH,
	)
}
