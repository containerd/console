package console

// #include <termios.h>
import "C"

import (
	"os"
	"syscall"
	"unsafe"
)

type master struct {
	f       *os.File
	termios *syscall.Termios
}

func (m *master) Read(b []byte) (int, error) {
	return m.f.Read(b)
}

func (m *master) Write(b []byte) (int, error) {
	return m.f.Write(b)
}

func (m *master) Close() error {
	return m.f.Close()
}

func (m *master) Resize(ws WinSize) error {
	if _, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		m.f.Fd(),
		uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	); err != 0 {
		return err
	}
	return nil
}

func (m *master) ResizeFrom(c Console) error {
	ws, err := c.Size()
	if err != nil {
		return err
	}
	return m.Resize(ws)
}

func (m *master) Reset() error {
	if m.termios == nil {
		return nil
	}
	return tcset(m.f.Fd(), m.termios)
}

func (m *master) SetRaw() error {
	m.termios = &syscall.Termios{}
	if err := tcget(m.f.Fd(), m.termios); err != 0 {
		return err
	}
	rawState := *m.termios
	C.cfmakeraw((*C.struct_termios)(unsafe.Pointer(&rawState)))
	rawState.Oflag = rawState.Oflag | C.OPOST
	if err := tcset(m.f.Fd(), &rawState); err != 0 {
		return err
	}
	return nil
}

func (m *master) Size() (WinSize, error) {
	var ws WinSize
	if _, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		m.f.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	); err != 0 {
		return ws, err
	}
	return ws, nil
}

func checkConsole(f *os.File) error {
	var termios syscall.Termios
	if tcget(f.Fd(), &termios) != 0 {
		return ErrNotAConsole
	}
	return nil
}
