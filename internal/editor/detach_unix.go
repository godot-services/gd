//go:build !windows

package editor

import "syscall"

func detachAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }
