package input

import (
	"slices"
	"testing"

	"github.com/Gaurav-Gosain/tuios/internal/config"
)

// TestUSLayoutKey pins what a key typed on a Cyrillic layout reads as, for every
// way a host can deliver it. The legacy and plain-text cases carry no base key,
// so the table is what answers them; the Kitty cases name the base key
// themselves.
func TestUSLayoutKey(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string // String() of the US equivalent
	}{
		{"legacy ц", "ц", "w"},
		{"legacy Ц", "Ц", "W"},
		{"legacy х", "х", "["},
		{"legacy Х", "Х", "{"},
		{"legacy ю", "ю", "."},
		{"legacy Б", "Б", "<"},
		{"legacy №", "№", "#"},
		{"legacy ukrainian і", "і", "s"},
		{"legacy alt+ц", "\x1bц", "alt+w"},
		{"legacy alt+Ц", "\x1bЦ", "alt+shift+w"},
		{"modifyOtherKeys ctrl+с", "\x1b[27;5;1089~", "ctrl+c"},
		{"kitty ctrl+с with base key", "\x1b[1089::99;5u", "ctrl+c"},
		{"kitty ctrl+с without base key", "\x1b[1089;5u", "ctrl+c"},
		{"kitty alt+ц with base key", "\x1b[1094::119;3u", "alt+w"},
		{"kitty alt+shift+ц", "\x1b[1094:1062:119;4u", "alt+shift+w"},
		{"kitty shift+н, all keys as escapes", "\x1b[1085:1053:121;2u", "Y"},
		{"kitty shift+3 typing №", "\x1b[51:8470;2;8470u", "#"},
		{"kitty ctrl+х", "\x1b[1093::91;5u", "ctrl+["},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := decodeKey(t, []byte(tt.raw))
			us, ok := usLayoutKey(msg)
			if !ok {
				t.Fatalf("usLayoutKey(%q) not translated (Code=%q Text=%q Mod=%v Base=%q)",
					tt.raw, msg.Code, msg.Text, msg.Mod, msg.BaseCode)
			}
			if got := us.String(); got != tt.want {
				t.Errorf("US key = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUSLayoutKeyLeavesLatinAlone guards the other half: a layout that types
// ASCII is bound by the character it types, so nothing it sends is translated,
// and neither is a key that has no character at all.
func TestUSLayoutKeyLeavesLatinAlone(t *testing.T) {
	for name, raw := range map[string]string{
		"plain w":                     "w",
		"plain W":                     "W",
		"alt+w":                       "\x1bw",
		"ctrl+w":                      "\x17",
		"AZERTY a on the q key":       "\x1b[97::113;2u",
		"AZERTY ctrl+a on the q key":  "\x1b[97::113;5u",
		"russian . on the / key":      ".",
		"up arrow":                    "\x1b[A",
		"accented é, typed as a char": "é",
	} {
		msg := decodeKey(t, []byte(raw))
		if us, ok := usLayoutKey(msg); ok {
			t.Errorf("%s: translated to %q, want untouched", name, us.String())
		}
	}
}

// TestBindingsAnswerOnCyrillicLayout is the report this exists for: with the
// default config, keys struck on a Russian layout reach the same actions as the
// Latin keys in the same place.
func TestBindingsAnswerOnCyrillicLayout(t *testing.T) {
	registry := config.NewKeybindRegistry(config.DefaultConfig())
	for _, tt := range []struct {
		latin, cyrillic string
		lookup          func(string) string
	}{
		{"t", "е", registry.GetAction},
		{"x", "ч", registry.GetAction},
		{"w", "ц", registry.GetPrefixAction},
		{"\x1b[110;3u", "\x1b[1090::110;3u", registry.GetTerminalModeAction},
		{"\x1bn", "\x1bт", registry.GetTerminalModeAction},
		{"\x1bN", "\x1bТ", registry.GetAction},
		{"\x1b[110:78;4u", "\x1b[1090:1058:110;4u", registry.GetAction},
	} {
		want := lookupAction(decodeKey(t, []byte(tt.latin)), tt.lookup)
		if want == "" {
			t.Fatalf("%q is not bound by default; pick a key that is", tt.latin)
		}
		got := lookupAction(decodeKey(t, []byte(tt.cyrillic)), tt.lookup)
		if got != want {
			t.Errorf("%q = %q, want %q (what %q does)", tt.cyrillic, got, want, tt.latin)
		}
	}
}

// TestBindingKeysLiteralFirst pins that a binding written on the Cyrillic letter
// itself still beats the Latin key it shares a place with.
func TestBindingKeysLiteralFirst(t *testing.T) {
	keys := bindingKeys(decodeKey(t, []byte("ц")))
	if len(keys) == 0 || keys[0] != "ц" {
		t.Fatalf("bindingKeys = %q, want the literal first", keys)
	}
	if !slices.Contains(keys, "w") {
		t.Errorf("bindingKeys = %q, want w among them", keys)
	}
}

// TestLeaderOnCyrillicLayout checks the leader is recognised whichever layout is
// active, including one bound to an Alt chord, which the legacy encoding sends
// as the Cyrillic letter.
func TestLeaderOnCyrillicLayout(t *testing.T) {
	for _, tt := range []struct{ leader, raw string }{
		{"ctrl+b", "\x1b[1080::98;5u"},
		{"ctrl+b", "\x02"},
		{"alt+a", "\x1bф"},
	} {
		s := &config.Settings{LeaderKey: tt.leader}
		if !isLeaderKey(decodeKey(t, []byte(tt.raw)), s) {
			t.Errorf("%q not recognised as leader %q", tt.raw, tt.leader)
		}
	}
}

// TestForwardNonLatinChords pins what a pane receives for the chords that used
// to arrive as nothing at all: under the Kitty protocol a Ctrl or Alt chord
// carries no text, and the code was only consulted when it was ASCII.
func TestForwardNonLatinChords(t *testing.T) {
	for _, tt := range []struct {
		name, raw, want string
	}{
		{"kitty ctrl+с interrupts", "\x1b[1089::99;5u", "\x03"},
		{"kitty ctrl+с without base key", "\x1b[1089;5u", "\x03"},
		{"modifyOtherKeys ctrl+с", "\x1b[27;5;1089~", "\x03"},
		{"kitty ctrl+х is ctrl+[", "\x1b[1093::91;5u", "\x1b"},
		{"kitty alt+ц", "\x1b[1094::119;3u", "\x1bц"},
		{"legacy alt+ц", "\x1bц", "\x1bц"},
		{"kitty alt+shift+a", "\x1b[97:65;4u", "\x1bA"},
		{"kitty alt+a", "\x1b[97;3u", "\x1ba"},
		{"legacy ctrl+c", "\x03", "\x03"},
		{"plain ц", "ц", "ц"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := string(getRawKeyBytesWithMode(decodeKey(t, []byte(tt.raw)), false))
			if got != tt.want {
				t.Errorf("forwarded %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRepairAlternateKeys pins the workaround for the decoder copying the base
// layout key into the shifted key. Without it shift+н, reported with all keys as
// escapes and no associated text, typed a lowercase y into the pane.
func TestRepairAlternateKeys(t *testing.T) {
	for _, tt := range []struct {
		name, raw, wantText string
		wantForward         string
	}{
		{"shift+н without associated text", "\x1b[1085:1053:121;2u", "Н", "Н"},
		{"shift+н with associated text", "\x1b[1085:1053:121;2;1053u", "Н", "Н"},
		{"AZERTY shift+a without associated text", "\x1b[97:65:113;2u", "A", "A"},
		{"alt+shift+ц", "\x1b[1094:1062:119;4u", "", "\x1bЦ"},
		{"plain н under all keys", "\x1b[1085::121u", "н", "н"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			msg := repairAlternateKeys(decodeKey(t, []byte(tt.raw)))
			if msg.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", msg.Text, tt.wantText)
			}
			if msg.ShiftedCode != 0 && msg.ShiftedCode == msg.BaseCode {
				t.Errorf("ShiftedCode is still the base key %q", msg.ShiftedCode)
			}
			if got := string(getRawKeyBytesWithMode(msg, false)); got != tt.wantForward {
				t.Errorf("forwarded %q, want %q", got, tt.wantForward)
			}
		})
	}
}

func TestDropLastRune(t *testing.T) {
	for in, want := range map[string]string{
		"":     "",
		"a":    "",
		"ab":   "a",
		"прив": "при",
		"aж":   "a",
		"é":    "",
	} {
		if got := dropLastRune(in); got != want {
			t.Errorf("dropLastRune(%q) = %q, want %q", in, got, want)
		}
	}
}
