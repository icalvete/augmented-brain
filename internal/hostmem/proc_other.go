//go:build !linux && !darwin

package hostmem

// Elsewhere (Windows) memory is not watched.
func local() System { return nil }
