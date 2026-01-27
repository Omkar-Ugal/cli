// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package types

import (
	"github.com/distribution/reference"
)

// ImageRef is a generic wrapper around a Docker image reference.
type ImageRef[T interface {
	reference.Reference
	comparable
}] struct {
	Reference T
}

func (ir ImageRef[T]) String() string {
	var zero T
	if ir.Reference == zero {
		return ""
	}
	return reference.FamiliarString(ir.Reference)
}

func (ir ImageRef[T]) MarshalText() ([]byte, error) {
	return []byte(ir.String()), nil
}
