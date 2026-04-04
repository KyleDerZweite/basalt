// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build windows

package cmd

import "syscall"

func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}
