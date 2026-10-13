// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package builder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	dockerconfig "github.com/docker/cli/cli/config"
	"github.com/moby/buildkit/client"
	gateway "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/progress/progresswriter"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/images"
	"unikraft.com/x/buildkit"
	ukio "unikraft.com/x/io"
	"unikraft.com/x/log"
)

// localBuildkit is the buildkitd that builds run on.
var localBuildkit = buildkit.Fallback{buildkit.Host{}, buildkit.Docker{}}

type buildkitContextKey struct{}

// withBuildkit makes the builds under ctx share d instead of acquiring their own.
func withBuildkit(ctx context.Context, d *buildkit.Daemon) context.Context {
	return context.WithValue(ctx, buildkitContextKey{}, d)
}

func buildkitFromContext(ctx context.Context) *buildkit.Daemon {
	d, _ := ctx.Value(buildkitContextKey{}).(*buildkit.Daemon)
	return d
}

// solver holds one BuildKit connection for several solves.
type solver struct {
	client  *client.Client
	release func() error
	session []session.Attachable
}

func newSolver(ctx context.Context) (*solver, error) {
	profile, err := config.G(ctx).CurrentProfile()
	if err != nil {
		return nil, err
	}
	dockerConfig := dockerconfig.LoadDefaultConfigFile(os.Stderr)

	release := func() error { return nil }
	d := buildkitFromContext(ctx)
	if d == nil {
		d, err = localBuildkit.Acquire(ctx, buildkit.Key{})
		if errors.Is(err, buildkit.ErrUnavailable) {
			log.G(ctx).Debug().Err(err).Msg("no buildkit")
			return nil, errors.New("buildkit not found; please start a docker or buildkit daemon, or set BUILDKIT_HOST")
		}
		if err != nil {
			return nil, err
		}
		log.G(ctx).Info().Str("addr", d.Addr).Msg("found buildkit")
		release = d.Release
	}

	return &solver{
		client:  d.Client,
		release: release,
		session: []session.Attachable{
			authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
				AuthConfigProvider: images.LoadBuildkitAuthConfig(dockerConfig, profile),
			}),
		},
	}, nil
}

func (s *solver) close() {
	_ = s.release()
}

// solveToTar runs build, exporting an uncompressed tarball of the result to
// dst, which is left synced and rewound. Progress is printed under prefix.
func (s *solver) solveToTar(ctx context.Context, dst *os.File, prefix string, solveOpt client.SolveOpt, build gateway.BuildFunc) error {
	solveOpt.Ref = identity.NewID()
	solveOpt.Session = s.session
	solveOpt.Exports = []client.ExportEntry{{
		Type: client.ExporterTar,
		Output: func(map[string]string) (io.WriteCloser, error) {
			return ukio.NopWriteCloser(dst), nil
		},
	}}

	pw, err := progresswriter.NewPrinter(context.WithoutCancel(ctx), os.Stderr, "auto")
	if err != nil {
		return err
	}
	w := progresswriter.NewMultiWriter(pw).WithPrefix(prefix, true)

	if _, err := s.client.Build(ctx, solveOpt, "buildctl", build, w.Status()); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-pw.Done():
	}
	if err := pw.Err(); err != nil {
		return err
	}

	if err := dst.Sync(); err != nil {
		return fmt.Errorf("could not sync tarball: %w", err)
	}
	if _, err := dst.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("could not rewind tarball: %w", err)
	}

	return nil
}
