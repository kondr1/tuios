package input

import (
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// cyrillicKeys maps what a Cyrillic layout types to the US QWERTY key in the
// same place, lowercase on both sides. Russian ЙЦУКЕН is the base; the
// Ukrainian and Belarusian letters sit on keys where Russian has a different
// one, so all three fit in one table without a clash.
//
// This is the fallback for a host that does not report the base layout key: a
// terminal without the Kitty protocol, or a plain letter under it, which is sent
// as text and so never carries the alternate-key subparameter.
var cyrillicKeys = map[rune]rune{
	'ё': '`',
	'й': 'q', 'ц': 'w', 'у': 'e', 'к': 'r', 'е': 't', 'н': 'y',
	'г': 'u', 'ш': 'i', 'щ': 'o', 'з': 'p', 'х': '[', 'ъ': ']',
	'ф': 'a', 'ы': 's', 'в': 'd', 'а': 'f', 'п': 'g', 'р': 'h',
	'о': 'j', 'л': 'k', 'д': 'l', 'ж': ';', 'э': '\'',
	'я': 'z', 'ч': 'x', 'с': 'c', 'м': 'v', 'и': 'b', 'т': 'n',
	'ь': 'm', 'б': ',', 'ю': '.',
	'і': 's', 'ї': ']', 'є': '\'', 'ў': 'o',
}

// cyrillicShiftedKeys holds the shifted characters of a Cyrillic layout that
// are not the capital of a letter, keyed to the unshifted US key they sit on.
var cyrillicShiftedKeys = map[rune]rune{
	'№': '3',
}

// usShifted is what Shift makes of each unshifted US QWERTY symbol key.
var usShifted = map[rune]rune{
	'`': '~', '1': '!', '2': '@', '3': '#', '4': '$', '5': '%', '6': '^',
	'7': '&', '8': '*', '9': '(', '0': ')', '-': '_', '=': '+',
	'[': '{', ']': '}', '\\': '|', ';': ':', '\'': '"',
	',': '<', '.': '>', '/': '?',
}

// repairAlternateKeys undoes a slip in ultraviolet's Kitty key decoder. When a
// report carries the base layout key, the decoder stores it as the shifted key
// too (its base-key case falls through into the shifted-key case), and when the
// report came without associated text it then derives the text from that. So on
// a Russian layout shift+н arrived as the text "y", typed into the pane as a
// lowercase Latin letter, and alt+shift+ц reached the pane as ESC w.
//
// Kitty only sends a base key that differs from the key itself, so a shifted
// code equal to it is the decoder's copy and not something the host said. The
// real shifted code is lost by then; dropping the copy is the honest answer,
// and the text is rebuilt the way the decoder would have built it from a right
// one. Idempotent, and a no-op for any event that carries no base key.
func repairAlternateKeys(msg tea.KeyPressMsg) tea.KeyPressMsg {
	if msg.BaseCode == 0 || msg.ShiftedCode != msg.BaseCode {
		return msg
	}
	msg.ShiftedCode = 0
	// Shift never makes a lowercase Latin letter, so a shifted key whose text is
	// its own lowercase base key is text built from the copy.
	if msg.Mod&tea.ModShift != 0 && msg.BaseCode >= 'a' && msg.BaseCode <= 'z' &&
		msg.Text == string(msg.BaseCode) {
		msg.Text = string(unicode.ToUpper(msg.Code))
	}
	return msg
}

// usLayoutKey returns msg as the same physical key would have arrived on a US
// QWERTY layout, and whether msg needed translating at all.
//
// Bindings are written in Latin, so on a Cyrillic layout none of them would
// ever match: w arrives as ц, ctrl+w as ctrl+ц, Y as Н. The translation only
// applies when the key produced a non-ASCII character. A layout that types
// ASCII (AZERTY, QWERTZ, Dvorak) is left alone, because there the character is
// the name the user binds by, and reading it as the key in the US position
// would move every one of their bindings.
func usLayoutKey(msg tea.KeyPressMsg) (tea.KeyPressMsg, bool) {
	nonLatin := func(r rune) bool { return r >= utf8.RuneSelf && unicode.IsPrint(r) }
	text := []rune(msg.Text)
	if !nonLatin(msg.Code) && (len(text) != 1 || !nonLatin(text[0])) {
		return tea.KeyPressMsg{}, false
	}

	mods := msg.Mod &^ lockMods
	shift := mods&tea.ModShift != 0
	var base rune
	switch {
	case msg.BaseCode > ' ' && msg.BaseCode < utf8.RuneSelf:
		// The Kitty protocol's alternate-key report names the PC-101 key, which
		// is exact where a table can only guess.
		base = unicode.ToLower(msg.BaseCode)
	case cyrillicKeys[unicode.ToLower(msg.Code)] != 0:
		base = cyrillicKeys[unicode.ToLower(msg.Code)]
		shift = shift || unicode.IsUpper(msg.Code)
	case cyrillicShiftedKeys[msg.Code] != 0:
		base = cyrillicShiftedKeys[msg.Code]
		shift = true
	case shift && msg.Code > ' ' && msg.Code < utf8.RuneSelf:
		// A digit row key whose shifted character is not ASCII, as № on a
		// Russian layout: the code already is the US key.
		base = msg.Code
	default:
		return tea.KeyPressMsg{}, false
	}

	us := tea.KeyPressMsg{Code: base, Mod: mods}
	char := base
	if shift {
		us.Mod |= tea.ModShift
		char = unicode.ToUpper(base)
		if s, ok := usShifted[base]; ok {
			char = s
		}
		us.ShiftedCode = char
	}
	// Text is what a bare or shifted key types. With any other modifier held
	// Bubble Tea leaves it empty, and the key goes by its keystroke name.
	if us.Mod&^tea.ModShift == 0 {
		us.Text = string(char)
	}
	return us, true
}

// hotkeyString is msg.String() for a key that is being read as a command, with a
// key typed on a non-Latin layout spelled as its US QWERTY equivalent. For the
// handlers that compare key names directly instead of going through the
// registry; anything that takes typed text has to keep reading msg itself.
func hotkeyString(msg tea.KeyPressMsg) string {
	if us, ok := usLayoutKey(msg); ok {
		return us.String()
	}
	return msg.String()
}

// dropLastRune is what backspace does to a query or a name: it removes the last
// character, not the last byte. Slicing off one byte left a Cyrillic or
// accented letter as half a UTF-8 sequence, which rendered as garbage and made
// every filter after it match nothing.
func dropLastRune(s string) string {
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}
