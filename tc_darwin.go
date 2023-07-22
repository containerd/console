/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package console

import (
	"bytes"
	"errors"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	cmdTcGet = unix.TIOCGETA
	cmdTcSet = unix.TIOCSETA
)

// unlockpt unlocks the slave pseudoterminal device corresponding to the master pseudoterminal referred to by f.
// unlockpt should be called before opening the slave side of a pty.
func unlockpt(f *os.File) error {
	return unix.IoctlSetPointerInt(int(f.Fd()), unix.TIOCPTYUNLK, 0)
}

// ptsname retrieves the name of the first available pts for the given master.
func ptsname(f *os.File) (string, error) {
	n := make([]byte, 128)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&n[0]))); errno != 0 {
		return "", errno
	}

	end := bytes.IndexByte(n, 0)
	if end < 0 {
		return "", errors.New("TIOCPTYGNAME string not NUL-terminated")
	}

	return string(n[:end]), nil
}
