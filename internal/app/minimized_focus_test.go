package app

import (
	"testing"

	"github.com/Gaurav-Gosain/tuios/internal/terminal"
)

// TestEnterTerminalModeRestoresAMinimizedPane is the regression test for typing
// into a pane that is not on screen.
//
// A minimized pane can hold the focus, because the rail and the window numbers
// address a pane by name rather than by what is visible. Entering terminal mode
// on one left the user typing into the dock: keystrokes reached the shell of a
// pane they could not see, and nothing on screen moved.
func TestEnterTerminalModeRestoresAMinimizedPane(t *testing.T) {
	for _, tiling := range []bool{false, true} {
		name := "floating"
		if tiling {
			name = "tiling"
		}
		t.Run(name, func(t *testing.T) {
			a := newTestWindow(t, "alpha-0001", 60, 20)
			b := newTestWindow(t, "bravo-0001", 60, 20)
			m := newTestOS(a)
			m.Windows = []*terminal.Window{a, b}
			m.Width, m.Height = 160, 40
			m.AutoTiling = tiling
			for _, w := range m.Windows {
				w.Workspace = m.CurrentWorkspace
			}

			m.MinimizeWindow(1)
			if !b.Minimized {
				t.Fatal("bravo did not minimize")
			}
			// What the rail does when its cursor activates a row: focus by
			// index, with no restore of its own.
			m.FocusWindow(1)
			if m.FocusedWindow != 1 {
				t.Fatalf("FocusedWindow = %d, want the minimized pane", m.FocusedWindow)
			}

			m.EnterTerminalMode()

			if b.Minimized {
				t.Error("entered terminal mode on a pane that is still minimized")
			}
			if m.Mode != TerminalMode {
				t.Errorf("Mode = %v, want TerminalMode", m.Mode)
			}
		})
	}
}
