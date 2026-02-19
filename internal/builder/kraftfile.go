// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package builder

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"unikraft.com/x/kraftfile"
)

func KraftfileToBuildOpts(dir string, kf *kraftfile.Kraftfile) (BuildOpts, error) {
	var opts BuildOpts

	opts.Cmd = []string(kf.Cmd)
	opts.Env = kf.Env
	opts.Labels = kf.Labels

	if kf.Runtime != nil {
		opts.Runtime = string(*kf.Runtime)
	}

	if kf.Unikraft != nil {
		return BuildOpts{}, fmt.Errorf("unikraft configuration not currently supported")
	}
	if kf.Libraries != nil {
		// these are the same build process as kf.Unikraft
		return BuildOpts{}, fmt.Errorf("library configuration not currently supported")
	}
	if kf.Volumes != nil {
		return BuildOpts{}, fmt.Errorf("volumes configuration not currently supported")
	}
	if kf.Template != nil {
		return BuildOpts{}, fmt.Errorf("template configuration not currently supported")
	}

	for _, target := range kf.Targets {
		features := make([]string, 0, len(target.KConfig))
		for _, kv := range target.KConfig {
			features = append(features, fmt.Sprintf("%s=%v", kv.Key, kv.Value))
		}
		version := fmt.Sprint(target.KConfig.Get("CONFIG_UK_FULLVERSION"))
		opts.Platform = append(opts.Platform, ocispec.Platform{
			Architecture: target.Arch,
			OS:           target.Plat,
			OSVersion:    version,
			OSFeatures:   features,
		})
	}

	// FIXME: import detection logic from kraftkit initrd/detect.go

	opts.Rootfs.Format = kf.Rootfs.Format
	base := filepath.Base(kf.Rootfs.Source)
	if base == "Dockerfile" || slices.Contains(strings.Split(base, "."), "Dockerfile") {
		opts.Rootfs.Path = filepath.Dir(kf.Rootfs.Source)
	} else {
		return BuildOpts{}, fmt.Errorf("unable to determine rootfs type for source %q", kf.Rootfs.Source)
	}

	opts.Rootfs.Path = filepath.Join(dir, opts.Rootfs.Path)

	opts.Rootfs.Compress = true

	return opts, nil
}
