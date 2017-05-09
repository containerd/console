// +build linux

package console

import "testing"

func TestWinSize(t *testing.T) {
	c, _, err := NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Resize(WinSize{
		Width:  11,
		Height: 10,
	}); err != nil {
		t.Error(err)
		return
	}
	size, err := c.Size()
	if err != nil {
		t.Error(err)
		return
	}
	if size.Width != 11 {
		t.Errorf("width should be 11 but received %d", size.Width)
	}
	if size.Height != 10 {
		t.Errorf("height should be 10 but received %d", size.Height)
	}
}
