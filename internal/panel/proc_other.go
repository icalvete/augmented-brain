//go:build !unix && !windows

package panel

import (
	"errors"
	"time"
)

func pidAlive(int) bool           { return false }
func exePath(int) (string, error) { return "", errors.New("not supported") }
func terminate(int) error         { return errors.New("not supported") }
func kill(int) error              { return errors.New("not supported") }
func procInfo(int) ([]string, time.Time, error) {
	return nil, time.Time{}, errors.New("not supported")
}

const exactExe = false
