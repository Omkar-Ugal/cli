// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package multimetro

import (
	"strings"

	"github.com/google/uuid"
	"unikraft.com/cloud/sdk/platform"
)

type Key struct {
	Metro string

	Name string
	UUID string
}

const MetroKeySeparator = "#"

func ParseKey(s string) Key {
	if metro, key, ok := strings.Cut(s, MetroKeySeparator); ok {
		if uuid.Validate(key) == nil {
			return Key{Metro: metro, UUID: key}
		}
		return Key{Metro: metro, Name: key}
	}

	if uuid.Validate(s) == nil {
		return Key{UUID: s}
	}
	return Key{Name: s}
}

func (k Key) NameOrUUID() platform.NameOrUUID {
	if k.UUID != "" {
		return platform.NameOrUUID{Uuid: &k.UUID}
	}
	return platform.NameOrUUID{Name: &k.Name}
}

func (k Key) String() string {
	s := ""
	if k.Metro != "" {
		s += k.Metro + MetroKeySeparator
	}
	if k.UUID != "" {
		s += k.UUID
	} else if k.Name != "" {
		s += k.Name
	}
	return s
}

type Keys []Key

func ParseKeys(ss []string) Keys {
	var keys []Key
	for _, s := range ss {
		keys = append(keys, ParseKey(s))
	}
	return keys
}

func (ks Keys) NamesOrUUIDs() []platform.NameOrUUID {
	var nou []platform.NameOrUUID
	for _, k := range ks {
		nou = append(nou, k.NameOrUUID())
	}
	return nou
}

func (ks Keys) Strings() []string {
	var ss []string
	for _, k := range ks {
		ss = append(ss, k.String())
	}
	return ss
}
