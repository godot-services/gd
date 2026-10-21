package editor

import "syscall"

const (
	detachedProcess       = 0x00000008
	createNewProcessGroup = 0x00000200
)

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcessGroup}
}
