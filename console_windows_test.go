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
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
)

// TestMaster_DelegatesToFile verifies that a master delegates Read, Write,
// Fd, and Name to the file it was constructed with, rather than the
// process's global os.Stdin/os.Stdout. The Write case is the regression
// test for https://github.com/containerd/console/issues/83, where output
// written to a console created from a non-stdout stream leaked to stdout.
//
// The master is backed by an os.Pipe so the assertions exercise real I/O
// without requiring an attached console, allowing the test to run anywhere.
func TestMaster_DelegatesToFile(t *testing.T) {
	t.Run("WriteGoesToFile", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		defer w.Close()

		want := []byte("written via master")

		// Drain the pipe concurrently so Write does not block.
		type result struct {
			data []byte
			err  error
		}
		done := make(chan result, 1)
		go func() {
			data, err := io.ReadAll(r)
			done <- result{data, err}
		}()

		m := &master{f: w}
		if _, err := m.Write(want); err != nil {
			t.Fatalf("Write: %v", err)
		}
		w.Close()

		got := <-done
		if got.err != nil {
			t.Fatalf("reading pipe: %v", got.err)
		}
		if string(got.data) != string(want) {
			t.Errorf("Write reached the wrong stream: pipe got %q, want %q", got.data, want)
		}
	})

	t.Run("ReadComesFromFile", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		defer w.Close()

		want := []byte("read via master")
		go func() {
			w.Write(want)
			w.Close()
		}()

		m := &master{f: r}
		got, err := io.ReadAll(m)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if string(got) != string(want) {
			t.Errorf("Read drew from the wrong stream: got %q, want %q", got, want)
		}
	})

	t.Run("FdAndName", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		defer w.Close()

		m := &master{f: w}
		if got, want := m.Fd(), w.Fd(); got != want {
			t.Errorf("Fd() = %v, want %v", got, want)
		}
		if m.Fd() == os.Stdout.Fd() {
			t.Error("Fd() returned os.Stdout's descriptor; expected the provided file's")
		}
		if got, want := m.Name(), w.Name(); got != want {
			t.Errorf("Name() = %q, want %q", got, want)
		}
	})
}

// TestConsoleFromFile_Delegation is an end-to-end test that exercises
// the public ConsoleFromFile API with real console handles.
//
// Because "go test" redirects standard streams to pipes, the test
// spawns itself as a subprocess with stdin/stdout connected to the real
// console devices (CONIN$/CONOUT$); the subprocess's stderr is captured
// so failures are reported with detail. This is the same helper-process
// pattern used in the Go standard library (os/exec tests).
func TestConsoleFromFile_Delegation(t *testing.T) {
	if os.Getenv("CONSOLE_E2E_SUBPROCESS") == "1" {
		// Subprocess: stdin and stdout are wired to the real console.
		// stderr is captured by the parent and used to report diagnostics.
		var failed bool
		for _, f := range []*os.File{os.Stdin, os.Stdout} {
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

	// stdin and stdout go to the real console (what we are testing); stderr is
	// captured so subprocess diagnostics surface in the failure message.
	var stderr bytes.Buffer
	cmd := exec.Command(os.Args[0], "-test.run=^TestConsoleFromFile_Delegation$")
	cmd.Env = append(os.Environ(), "CONSOLE_E2E_SUBPROCESS=1")
	cmd.Stdin = conin
	cmd.Stdout = conout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("subprocess failed: %v\n%s", err, stderr.String())
	}
}
