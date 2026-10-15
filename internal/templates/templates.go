// Package templates holds the files a new space starts with, in every supported
// language. They are embedded in the binary.
package templates

import "embed"

// FS has one folder per language: profile.md, and space/ with the files of a new space.
//
//go:embed en es
var FS embed.FS
