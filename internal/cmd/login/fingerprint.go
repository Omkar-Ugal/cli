// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package login

import (
	"context"
	"os/exec"
	"runtime"
	"strings"

	"tailscale.com/hostinfo"
	"tailscale.com/util/dnsname"
	"unikraft.com/cli/internal/version"
	"unikraft.com/cloud/sdk/controlplane"
	"unikraft.com/x/log"
	"unikraft.com/x/ptr"
)

func getFingerprint(ctx context.Context) controlplane.RequestSigninRequest {
	host := hostinfo.New()
	container, _ := host.Container.Get()

	if runtime.GOOS == "darwin" {
		var err error
		host.OSVersion, err = getMacOSVersion()
		if err != nil {
			log.G(ctx).
				Warn().
				Err(err).
				Msg("failed to get macOS version")
		}
	}

	return controlplane.RequestSigninRequest{
		Hostname:       ptr.ToPtr(dnsname.TrimCommonSuffixes(host.Hostname)),
		Os:             &host.OS,
		Container:      &container,
		Distro:         &host.Distro,
		DistroCodename: &host.DistroCodeName,
		DistroVersion:  &host.DistroVersion,
		CliVersion:     &version.Version,
		Goarch:         ptr.ToPtr(runtime.GOARCH),
		Goos:           ptr.ToPtr(runtime.GOOS),
		GoVersion:      ptr.ToPtr(runtime.Version()),
		OsVersion:      &host.OSVersion,
	}
}

func getMacOSVersion() (string, error) {
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(output))
	return version, nil
}
