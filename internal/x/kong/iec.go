// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package kong

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
)

type IEC string

var reIEC = regexp.MustCompile(`(?i)^\s*(\d+)\s*(kib|mib|gib|kb|mb|gb)?\s*$`)

// Decode implements the kong.Decoder interface to parse IEC size strings.
func (m *IEC) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	var raw string
	err := ctx.Scan.PopValueInto("string", &raw)
	if err != nil {
		return err
	}

	mb, err := m.Value()
	if err != nil {
		return err
	}

	target.SetUint(mb)

	return nil
}

func (m *IEC) Value() (uint64, error) {
	matches := reIEC.FindStringSubmatch(string(*m))
	if matches == nil {
		return 0, fmt.Errorf("invalid size %s (expected IEC value e.g. 128M, 1GiB)", string(*m))
	}

	value, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, err
	}

	unit := strings.ToLower(matches[2])

	var bytes uint64
	switch unit {
	case "", "mb", "m":
		bytes = value * 1_000_000

	case "kb", "k":
		bytes = value * 1_000

	case "gb", "g":
		bytes = value * 1_000_000_000

	case "kib":
		bytes = value * 1024

	case "mib":
		bytes = value * 1024 * 1024

	case "gib":
		bytes = value * 1024 * 1024 * 1024

	default:
		return 0, fmt.Errorf("unknown unit %q", unit)
	}

	return bytes / 1_000_000, nil
}
