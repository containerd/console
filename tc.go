// +build linux

package console

import (
	"syscall"
	"unsafe"
)

// #include <termios.h>
import "C"

func tcget(fd uintptr, p *syscall.Termios) syscall.Errno {
	ret, err := C.tcgetattr(C.int(fd), (*C.struct_termios)(unsafe.Pointer(p)))
	if ret != 0 {
		return err.(syscall.Errno)
	}
	return 0
}

func tcset(fd uintptr, p *syscall.Termios) syscall.Errno {
	ret, err := C.tcsetattr(C.int(fd), C.TCSANOW, (*C.struct_termios)(unsafe.Pointer(p)))
	if ret != 0 {
		return err.(syscall.Errno)
	}
	return 0
}
