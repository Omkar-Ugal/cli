// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package version

import (
	"fmt"

	"unikraft.com/cli/internal/config"
)

type VersionCmd struct{}

func (cmd VersionCmd) Run(stdio config.Stdio) error {
	fmt.Fprintln(stdio.Stdout, String())
	return nil
}
