// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The KraftKit Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package erofs

import (
	"context"
	"io"
	"os"

	"github.com/unikraft/go-archivefs/erofs"
)

var WithAllRoot = erofs.WithAllRoot

// CreateFSFromDirectory creates an EroFS filesystem from a directory.
func CreateFSFromDirectory(ctx context.Context, w io.WriterAt, source string, opts ...erofs.ErofsCreateOption) error {
	return erofs.Create(w, os.DirFS(source), opts...)
}
