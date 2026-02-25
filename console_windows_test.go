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
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// TestNewMaster_FileDelegation verifies that the master struct delegates
// Fd() and Name() to the file it was constructed with.
// This directly tests the struct so it works even when standard streams
// are redirected (e.g. inside "go test" or CI).
func TestNewMaster_FileDelegation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file *os.File
	}{
		{"stdin", os.Stdin},
		{"stdout", os.Stdout},
		{"stderr", os.Stderr},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &master{f: tt.file}
			m.initStdios()

			if got, want := m.Fd(), tt.file.Fd(); got != want {
				t.Errorf("Fd() = %v, want %v", got, want)
			}
			if got, want := m.Name(), tt.file.Name(); got != want {
				t.Errorf("Name() = %q, want %q", got, want)
			}
		})
	}
}

// TestConsoleFromFile_Delegation is an end-to-end test that exercises
// the public ConsoleFromFile API with real console handles.
//
// Because "go test" redirects standard streams to pipes, the test
// spawns itself as a subprocess with stdin/stdout/stderr connected
// to the real console devices (CONIN$/CONOUT$). This is the same
// helper-process pattern used in the Go standard library (os/exec tests).
func TestConsoleFromFile_Delegation(t *testing.T) {
	if os.Getenv("CONSOLE_E2E_SUBPROCESS") == "1" {
		// Subprocess: std handles are real console handles.
		var failed bool
		for _, f := range []*os.File{os.Stdin, os.Stdout, os.Stderr} {
			c, err := ConsoleFromFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ConsoleFromFile(%s): %v\n", f.Name(), err)
				failed = true
				continue
			}
			if got, want := c.Fd(), f.Fd(); got != want {
				fmt.Fprintf(os.Stderr, "ConsoleFromFile(%s).Fd() = %v, want %v\n", f.Name(), got, want)
				failed = true
			}
			if got, want := c.Name(), f.Name(); got != want {
				fmt.Fprintf(os.Stderr, "ConsoleFromFile(%s).Name() = %q, want %q\n", f.Name(), got, want)
				failed = true
			}
		}
		if failed {
			os.Exit(1)
		}
		os.Exit(0)
	}

	conin, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		t.Skip("no console available:", err)
	}
	defer conin.Close()

	conout, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		t.Skip("no console available:", err)
	}
	defer conout.Close()

	cmd := exec.Command(os.Args[0], "-test.run=^TestConsoleFromFile_Delegation$")
	cmd.Env = append(os.Environ(), "CONSOLE_E2E_SUBPROCESS=1")
	cmd.Stdin = conin
	cmd.Stdout = conout
	cmd.Stderr = conout

	if err := cmd.Run(); err != nil {
		t.Fatalf("subprocess failed: %v", err)
	}
}
