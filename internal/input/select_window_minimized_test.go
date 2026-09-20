package input

import (
	"testing"

	"github.com/Gaurav-Gosain/tuios/internal/app"
	"github.com/Gaurav-Gosain/tuios/internal/config"
	"github.com/Gaurav-Gosain/tuios/internal/terminal"
)

// selectWindowOS is three panes on one workspace, the middle one minimized.
func selectWindowOS(t *testing.T, tiling bool) *app.OS {
	t.Helper()
	wins := make([]*terminal.Window, 0, 3)
	for _, id := range []string{"alpha-0001", "bravo-0001", "carol-0001"} {
		ch := make(chan struct{}, 1)
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-ch:
				case <-done:
					return
				}
			}
		}()
		t.Cleanup(func() { close(done) })
		win := terminal.NewDaemonWindow(id, "test", 0, 0, 40, 10, 0, "pty-"+id, ch, config.DefaultScrollbackLines)
		if win == nil {
			t.Fatalf("NewDaemonWindow(%s) returned nil", id)
		}
		t.Cleanup(func() { win.Close() })
		win.Workspace = 1
		wins = append(wins, win)
	}
	o := &app.OS{
		Settings:         config.Global,
		Windows:          wins,
		FocusedWindow:    0,
		WorkspaceFocus:   map[int]int{},
		NumWorkspaces:    9,
		CurrentWorkspace: 1,
		Width:            160,
		Height:           40,
		AutoTiling:       tiling,
	}
	o.MinimizeWindow(1)
	return o
}

// TestSelectWindowByIndexSkipsMinimized pins that the window numbers address
// what is on screen. With tiling off they used to count the dock as well, so
// the number focused a pane the user could not see, and the next key typed went
// into it.
func TestSelectWindowByIndexSkipsMinimized(t *testing.T) {
	for _, tiling := range []bool{false, true} {
		name := "floating"
		if tiling {
			name = "tiling"
		}
		t.Run(name, func(t *testing.T) {
			o := selectWindowOS(t, tiling)
			if !o.Windows[1].Minimized {
				t.Fatal("bravo did not minimize")
			}

			// 2 is the second pane still on screen, which is carol.
			selectWindowByIndex(2, o)
			if got := o.FocusedWindow; got != 2 {
				t.Errorf("window 2 focused pane %d, want carol (2)", got)
			}
			if o.Windows[o.FocusedWindow].Minimized {
				t.Error("focused a minimized pane")
			}

			// There is no third pane on screen, so the number is a no-op
			// rather than a way into the dock.
			selectWindowByIndex(3, o)
			if o.Windows[o.FocusedWindow].Minimized {
				t.Error("window 3 focused a minimized pane")
			}
		})
	}
}
