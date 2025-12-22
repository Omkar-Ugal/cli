// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package multimetro

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/log"

	"unikraft.com/cli/internal/config"
)

type Client struct {
	clients    []*MetroClient
	clientsMap map[string]*MetroClient
}

type MetroClient struct {
	platform.Client
	Metro config.Metro
}

func NewClient(ctx context.Context) (*Client, error) {
	cfg := config.FromContextOrDefault(ctx)
	profile, err := cfg.CurrentProfile()
	if err != nil {
		return nil, err
	}
	if len(profile.Metros) == 0 {
		return nil, fmt.Errorf("profile %q has no metros configured", profile.Name)
	}
	metros := profile.Metros

	metroNames := make([]string, 0, len(metros))
	for _, metro := range metros {
		metroNames = append(metroNames, metro.Name)
	}
	log.G(ctx).
		Trace().
		Strs("metros", metroNames).
		Msg("initializing platform clients")

	clients := make([]*MetroClient, 0, len(metros))
	clientMap := make(map[string]*MetroClient, len(metros))
	for _, metro := range metros {
		client := platform.NewClient(
			platform.WithToken(profile.Token),
			platform.WithDefaultMetro(metro.Endpoint),
		)

		metroClient := &MetroClient{
			Client: client,
			Metro:  metro,
		}
		clients = append(clients, metroClient)
		clientMap[metro.Name] = metroClient
	}

	return &Client{
		clients:    clients,
		clientsMap: clientMap,
	}, nil
}

func (c *Client) getByMetro(metro string) (*MetroClient, error) {
	client, ok := c.clientsMap[metro]
	if !ok {
		return nil, fmt.Errorf("unknown metro %q", metro)
	}
	return client, nil
}

func (c *Client) getByKey(key Key) (*MetroClient, error) {
	if key.Metro != "" {
		return c.getByMetro(key.Metro)
	}
	return nil, fmt.Errorf("metro not specified for key")
}

func DoAll[T any](ctx context.Context, c *Client, fn func(context.Context, *MetroClient) (T, error)) ([]T, error) {
	eg, ctx := errgroup.WithContext(ctx)
	results := make([]T, len(c.clients))
	for idx, client := range c.clients {
		eg.Go(func() error {
			logger := log.FromContextOrDefault(ctx).
				With().
				Str("metro", client.Metro.Name).
				Logger()
			ctx := log.WithLogger(ctx, &logger)

			v, err := fn(ctx, client)
			if err != nil {
				return err
			}
			results[idx] = v
			return nil
		})
	}
	return results, eg.Wait()
}

func DoMetro[T any](ctx context.Context, c *Client, metro string, fn func(context.Context, *MetroClient) (T, error)) (T, error) {
	metroClient, err := c.getByMetro(metro)
	if err != nil {
		var zero T
		return zero, err
	}

	logger := log.FromContextOrDefault(ctx).
		With().
		Str("metro", metroClient.Metro.Name).
		Logger()
	ctx = log.WithLogger(ctx, &logger)

	return fn(ctx, metroClient)
}

func DoKeyExact[T any](ctx context.Context, c *Client, key Key, fn func(context.Context, *MetroClient) (T, error)) (T, error) {
	metroClient, err := c.getByKey(key)
	if err != nil {
		var zero T
		return zero, err
	}

	logger := log.FromContextOrDefault(ctx).
		With().
		Str("metro", metroClient.Metro.Name).
		Str("key", key.String()).
		Logger()
	ctx = log.WithLogger(ctx, &logger)

	return fn(ctx, metroClient)
}

func DoKeys[T any](ctx context.Context, c *Client, keys Keys, fn func(context.Context, *MetroClient, Keys) ([]T, []Key, error)) ([]T, error) {
	targets := make(map[*MetroClient]Keys)
	for _, key := range keys {
		if key.Metro != "" {
			client, ok := c.clientsMap[key.Metro]
			if !ok {
				return nil, fmt.Errorf("unknown metro %q", key.Metro)
			}
			targets[client] = append(targets[client], key)
		} else {
			for _, client := range c.clients {
				targets[client] = append(targets[client], key)
			}
		}
	}

	eg, ctx := errgroup.WithContext(ctx)
	foundKeys := make(map[Key]struct{})
	foundValues := make([]T, 0)
	var mu sync.Mutex

	for _, client := range c.clients {
		keys, ok := targets[client]
		if !ok || len(keys) == 0 {
			continue
		}

		eg.Go(func() error {
			logger := log.FromContextOrDefault(ctx).
				With().
				Str("metro", client.Metro.Name).
				Strs("keys", keys.Strings()).
				Logger()
			ctx := log.WithLogger(ctx, &logger)

			vals, keys, err := fn(ctx, client, keys)
			if err != nil {
				return err
			}

			mu.Lock()
			for _, key := range keys {
				foundKeys[key] = struct{}{}
				foundKeys[Key{UUID: key.UUID}] = struct{}{}
				foundKeys[Key{Name: key.Name}] = struct{}{}
				foundKeys[Key{Metro: client.Metro.Name, Name: key.Name}] = struct{}{}
				foundKeys[Key{Metro: client.Metro.Name, UUID: key.UUID}] = struct{}{}
				foundKeys[Key{Name: key.Name, UUID: key.UUID}] = struct{}{}
			}
			foundValues = append(foundValues, vals...)
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	notFound := make([]string, 0)
	for _, key := range keys {
		if _, ok := foundKeys[key]; !ok {
			notFound = append(notFound, key.String())
		}
	}
	if len(notFound) > 0 {
		return nil, fmt.Errorf("keys not found: %v", notFound)
	}

	return foundValues, nil
}
