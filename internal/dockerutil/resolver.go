// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package dockerutil

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/containerd/containerd/v2/core/remotes"
	"github.com/containerd/containerd/v2/core/remotes/docker"
	dockerconfig "github.com/docker/cli/cli/config"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/version"
)

func Resolver(profile *config.Profile) remotes.Resolver {
	headers := http.Header{}
	headers.Set("User-Agent", version.UserAgent())

	dockerConfig := dockerconfig.LoadDefaultConfigFile(os.Stderr)
	opts := []docker.RegistryOpt{
		docker.WithAuthorizer(docker.NewDockerAuthorizer(docker.WithAuthCreds(func(hostname string) (string, string, error) {
			if profile != nil {
				// FIXME: why are there two different auth schemes?
				if hostname == "index.unikraft.io" {
					return decodeAuth(profile.Token)
				}
				for _, metro := range profile.Metros {
					if hostname == metro.Index() {
						username := profile.Organization
						if username == "" {
							// organization may not be set on old or manually created
							// profiles - so fall back to decoding the username from the
							// token itself
							username, _, _ = decodeAuth(profile.Token)
						}
						return username, profile.Token, nil
					}
				}
			}

			auth, err := dockerConfig.GetCredentialsStore(hostname).Get(hostname)
			if err != nil {
				return "", "", err
			}
			if auth.IdentityToken != "" {
				return "", auth.IdentityToken, nil
			}
			return auth.Username, auth.Password, nil
		},
		))),
	}
	opts = append(opts, docker.WithPlainHTTP(docker.MatchAllHosts))

	dro := docker.ResolverOptions{
		Headers: headers,
		Hosts: fallbackHost(
			docker.ConfigureDefaultRegistries(opts...),
			docker.ConfigureDefaultRegistries(append(opts, docker.WithHostTranslator(func(s string) (string, error) {
				for _, metro := range profile.Metros {
					if s == metro.Index() {
						return "index.unikraft.io", nil
					}
				}
				return s, nil
			}))...),
		),
	}
	return docker.NewResolver(dro)
}

// decodeAuth is imported from github.com/docker/cli/cli/config/configfile, and
// is the same logic used to decode the "auth" field in the Docker config file.
func decodeAuth(authStr string) (string, string, error) {
	if authStr == "" {
		return "", "", nil
	}

	decLen := base64.StdEncoding.DecodedLen(len(authStr))
	decoded := make([]byte, decLen)
	authByte := []byte(authStr)
	n, err := base64.StdEncoding.Decode(decoded, authByte)
	if err != nil {
		return "", "", err
	}
	if n > decLen {
		return "", "", errors.New("something went wrong decoding auth config")
	}
	userName, password, ok := strings.Cut(string(decoded), ":")
	if !ok || userName == "" {
		return "", "", errors.New("invalid auth configuration file")
	}
	return userName, strings.Trim(password, "\x00"), nil
}

func fallbackHost(registryHosts ...docker.RegistryHosts) docker.RegistryHosts {
	return func(host string) ([]docker.RegistryHost, error) {
		var allHosts []docker.RegistryHost
		for _, registryHost := range registryHosts {
			hosts, err := registryHost(host)
			if err != nil {
				return nil, err
			}
			allHosts = append(allHosts, hosts...)
		}
		return allHosts, nil
	}
}
