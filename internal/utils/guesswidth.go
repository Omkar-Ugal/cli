//go:build appengine || (!linux && !freebsd && !darwin && !dragonfly && !netbsd && !openbsd)
// +build appengine !linux,!freebsd,!darwin,!dragonfly,!netbsd,!openbsd

package utils

import "io"

// GuessTermWidth returns a default terminal width of 80 characters since the
// environment does not support querying terminal dimensions.
func GuessTermWidth(w io.Writer) int {
	return 80
}
