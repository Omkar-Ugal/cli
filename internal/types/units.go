// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package types

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/docker/go-units"
)

// DurationMS is a time wrapper that represents a duration in milliseconds.
type DurationMS int64

func (d DurationMS) Unwrap() any {
	return time.Duration(d) * time.Millisecond
}

// DurationUS is a time wrapper that represents a duration in microseconds.
type DurationUS int64

func (d DurationUS) Unwrap() any {
	return time.Duration(d) * time.Microsecond
}

// SizeMebibytes is a size wrapper that represents a size in mebibytes.
type SizeMebibytes int64

func (m *SizeMebibytes) UnmarshalText(text []byte) error {
	if d, err := strconv.Atoi(string(text)); err == nil {
		*m = SizeMebibytes(d)
		return nil
	}
	size, err := units.RAMInBytes(string(text))
	if err != nil {
		return err
	}
	*m = SizeMebibytes(size / units.MiB)
	return nil
}

func (m *SizeMebibytes) UnmarshalJSON(data []byte) error {
	if len(data) != 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		return m.UnmarshalText([]byte(text))
	}
	var size int64
	if err := json.Unmarshal(data, &size); err != nil {
		return err
	}
	*m = SizeMebibytes(size)
	return nil
}

func (m SizeMebibytes) MarshalText() ([]byte, error) {
	return []byte(units.BytesSize(float64(m) * units.MiB)), nil
}

func (m SizeMebibytes) MarshalJSON() ([]byte, error) {
	text, err := m.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(text))
}

func (m SizeMebibytes) String() string {
	return units.BytesSize(float64(m) * units.MiB)
}
