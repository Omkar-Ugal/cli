// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"fmt"

	"unikraft.com/cli/internal/resource"
)

func getFromListable(ctx context.Context, listable resource.ListableResource, keys []string) ([]resource.Resource, error) {
	all, err := listable.List(ctx)
	if err != nil {
		return nil, err
	}

	found := make([]resource.Resource, 0, len(keys))
	var notFound []string
loop:
	for _, key := range keys {
		for _, resource := range all {
			if resource.Key() == key {
				found = append(found, resource)
				continue loop
			}
		}
		notFound = append(notFound, key)
	}

	if len(notFound) == 1 {
		return nil, fmt.Errorf("%s not found: %s", listable.Type().Name, notFound)
	} else if len(notFound) > 0 {
		return nil, fmt.Errorf("%s not found: %s", listable.Type().Names, notFound)
	}
	return found, nil
}
