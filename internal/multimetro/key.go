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

const (
	MetroKeySeparator = "/"
	KeyNamePrefix     = "name:"
	KeyUUIDPrefix     = "uuid:"
)

func ParseKey(s string) Key {
	metro := ""
	key := s
	if metroPart, keyPart, ok := strings.Cut(s, MetroKeySeparator); ok {
		metro = metroPart
		key = keyPart
	}

	name, id := parseKeyValue(key)
	return Key{Metro: metro, Name: name, UUID: id}
}

func parseKeyValue(key string) (name string, id string) {
	if name, ok := strings.CutPrefix(key, KeyNamePrefix); ok {
		return name, ""
	}
	if id, ok := strings.CutPrefix(key, KeyUUIDPrefix); ok {
		return "", id
	}
	if uuid.Validate(key) == nil {
		return "", key
	}
	return key, ""
}

func (k Key) NameOrUUID() platform.NameOrUUID {
	if k.UUID != "" {
		return platform.NameOrUUID{Uuid: &k.UUID}
	}
	return platform.NameOrUUID{Name: &k.Name}
}

func requiresNamePrefix(name string) bool {
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, KeyNamePrefix) || strings.HasPrefix(name, KeyUUIDPrefix) {
		return true
	}
	return uuid.Validate(name) == nil
}

func requiresIDPrefix(id string) bool {
	if id == "" {
		return false
	}
	if strings.HasPrefix(id, KeyNamePrefix) || strings.HasPrefix(id, KeyUUIDPrefix) {
		return true
	}
	return uuid.Validate(id) != nil
}

func (k Key) String() string {
	s := ""
	if k.Metro != "" {
		s += k.Metro + MetroKeySeparator
	}
	if k.UUID != "" {
		if requiresIDPrefix(k.UUID) {
			s += KeyUUIDPrefix
		}
		s += k.UUID
	} else if k.Name != "" {
		if requiresNamePrefix(k.Name) {
			s += KeyNamePrefix
		}
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
