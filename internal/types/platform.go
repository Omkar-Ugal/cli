// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package types

import "github.com/containerd/platforms"

// Platform wraps containerd platform values.
type Platform platforms.Platform

func (p Platform) String() string {
	return platforms.Format(platforms.Platform(p))
}

func (p Platform) MarshalText() ([]byte, error) {
	return []byte(platforms.Format(platforms.Platform(p))), nil
}

func (p *Platform) UnmarshalText(text []byte) error {
	plat, err := platforms.Parse(string(text))
	if err != nil {
		return err
	}
	*p = Platform(plat)
	return nil
}
