// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2024, Unikraft GmbH and The KraftKit Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cpio

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/unikraft/go-cpio"
	"unikraft.com/x/log"
)

type createOptions struct {
	allRoot bool
}

type CpioCreateOption func(*createOptions)

// WithAllRoot toggles whether all files permissions should be set to root:root
// instead of the original file permissions.
func WithAllRoot(allRoot bool) CpioCreateOption {
	return func(co *createOptions) {
		co.allRoot = allRoot
	}
}

// CreateFSFromDirectory creates a CPIO filesystem from an existing directory.
func CreateFSFromDirectory(ctx context.Context, w io.Writer, source string, opts ...CpioCreateOption) error {
	c := createOptions{}
	for _, opt := range opts {
		opt(&c)
	}

	cw := cpio.NewWriter(w)

	type hardlinkKey struct {
		device int
		inode  int64
	}
	hardlinks := map[hardlinkKey]string{}

	// Recursively walk and serialize to the output
	if err := filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("received error before parsing path: %w", err)
		}

		internal := strings.TrimPrefix(path, filepath.Clean(source))
		if internal == "" {
			return nil // Do not archive empty paths
		}
		internal = "." + filepath.ToSlash(internal)

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("could not get directory entry info: %w", err)
		}

		if d.Type().IsDir() {
			header := &cpio.Header{
				Name:    internal,
				Mode:    cpio.FileMode(info.Mode().Perm()) | cpio.TypeDir,
				ModTime: info.ModTime(),
				Size:    0, // Directories have size 0 in cpio
			}

			// Populate platform specific information
			FileInfoToCPIOHeader(info, header)

			if err := cw.WriteHeader(header); err != nil {
				return fmt.Errorf("could not write CPIO header: %w", err)
			}
			return nil
		}

		log.G(ctx).
			Debug().
			Str("file", internal).
			Msg("archiving")

		header := &cpio.Header{
			Name:    internal,
			Mode:    cpio.FileMode(info.Mode().Perm()),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}

		// Populate platform specific information
		FileInfoToCPIOHeader(info, header)

		if c.allRoot {
			header.Uid = 0
			header.Guid = 0
		}

		var data []byte
		switch {
		case info.Mode().IsRegular():
			header.Mode |= cpio.TypeRegular
			isHardlink := header.Links > 1 && header.Inode != 0
			if isHardlink {
				key := hardlinkKey{device: header.DeviceID, inode: header.Inode}
				if _, ok := hardlinks[key]; ok {
					header.Size = 0
					break
				}
				hardlinks[key] = internal
			}

			data, err = os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("could not read file: %w", err)
			}
			header.Size = int64(len(data))

		case info.Mode()&fs.ModeSymlink != 0:
			targetLink, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("could not read symlink: %w", err)
			}
			data = []byte(targetLink)
			header.Mode |= cpio.TypeSymlink
			header.Linkname = targetLink
			header.Size = int64(len(data))

		default:
			log.G(ctx).Warn().Msgf("unsupported file: %s", path)
			return nil
		}

		if err := cw.WriteHeader(header); err != nil {
			return fmt.Errorf("writing cpio header for %q: %w", internal, err)
		}

		if len(data) > 0 {
			if _, err := cw.Write(data); err != nil {
				return fmt.Errorf("could not write CPIO data for %s: %w", internal, err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("could not walk output path: %w", err)
	}

	return nil
}
