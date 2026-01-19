// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package reference

import (
	"github.com/distribution/reference"
)

// MatchNamed compares a reference.Named against a pattern string.
func MatchNamed(ref reference.Named, pattern string) bool {
	spec, err := reference.Parse(pattern)
	if err != nil {
		return false
	}
	specNamed, ok := spec.(reference.Named)
	if !ok {
		return false
	}

	if specNamed.Name() != ref.Name() {
		if specNamed.Name() != reference.Path(ref) {
			return false
		}
	}

	if digested, ok := specNamed.(reference.Digested); ok {
		n, ok := ref.(reference.Digested)
		if !ok || digested.Digest() != n.Digest() {
			return false
		}
		return true
	}
	if tagged, ok := specNamed.(reference.Tagged); ok {
		n, ok := ref.(reference.Tagged)
		if !ok || tagged.Tag() != n.Tag() {
			return false
		}
	}

	return true
}
