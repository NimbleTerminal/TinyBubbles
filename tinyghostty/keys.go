package tinyghostty

import (
	"strconv"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// AppendKey appends the xterm encoding of k to dst. appCursor selects SS3
// cursor keys (DECCKM). Unknown keys append nothing. The Kitty keyboard
// protocol is not supported.
func AppendKey(dst []byte, k tea.Key, appCursor bool) []byte {
	mod := k.Mod & (tea.ModShift | tea.ModAlt | tea.ModCtrl | tea.ModMeta)
	// xterm modifier parameter: 1 + shift + 2·alt + 4·ctrl + 8·meta.
	param := 1
	for i, m := range []tea.KeyMod{tea.ModShift, tea.ModAlt, tea.ModCtrl, tea.ModMeta} {
		if mod&m != 0 {
			param += 1 << i
		}
	}
	switch k.Code {
	case tea.KeyUp:
		return cursorKey(dst, 'A', param, appCursor)
	case tea.KeyDown:
		return cursorKey(dst, 'B', param, appCursor)
	case tea.KeyRight:
		return cursorKey(dst, 'C', param, appCursor)
	case tea.KeyLeft:
		return cursorKey(dst, 'D', param, appCursor)
	case tea.KeyBegin:
		return cursorKey(dst, 'E', param, appCursor)
	case tea.KeyEnd:
		return cursorKey(dst, 'F', param, appCursor)
	case tea.KeyHome:
		return cursorKey(dst, 'H', param, appCursor)
	case tea.KeyInsert:
		return tilde(dst, 2, param)
	case tea.KeyDelete:
		return tilde(dst, 3, param)
	case tea.KeyPgUp:
		return tilde(dst, 5, param)
	case tea.KeyPgDown:
		return tilde(dst, 6, param)
	case tea.KeyF1, tea.KeyF2, tea.KeyF3, tea.KeyF4:
		final := byte('P' + k.Code - tea.KeyF1)
		if param == 1 {
			return append(dst, 0x1b, 'O', final)
		}
		return append(strconv.AppendInt(append(dst, "\x1b[1;"...), int64(param), 10), final)
	case tea.KeyF5, tea.KeyF6, tea.KeyF7, tea.KeyF8, tea.KeyF9, tea.KeyF10, tea.KeyF11, tea.KeyF12:
		codes := [...]int{15, 17, 18, 19, 20, 21, 23, 24}
		return tilde(dst, codes[k.Code-tea.KeyF5], param)
	case tea.KeyTab:
		if mod&tea.ModShift != 0 {
			return append(dst, "\x1b[Z"...)
		}
		return alt(dst, mod, '\t')
	case tea.KeyEnter, tea.KeyKpEnter:
		return alt(dst, mod, '\r')
	case tea.KeyBackspace:
		if mod&tea.ModCtrl != 0 {
			return alt(dst, mod, 0x08)
		}
		return alt(dst, mod, 0x7f)
	case tea.KeyEscape:
		return alt(dst, mod, 0x1b)
	}
	r := k.Code
	if mod&tea.ModCtrl != 0 {
		switch {
		case r >= 'a' && r <= 'z':
			return alt(dst, mod, byte(r-'a'+1))
		case r == ' ' || r == '@' || r == '2':
			return alt(dst, mod, 0)
		case r >= '[' && r <= '_':
			return alt(dst, mod, byte(r-'['+0x1b))
		}
	}
	if k.Text == "" {
		if mod&tea.ModShift != 0 && k.ShiftedCode != 0 {
			r = k.ShiftedCode
		}
		// Special keys sit above utf8.MaxRune.
		if r < 0x20 || r == 0x7f || !utf8.ValidRune(r) {
			return dst
		}
	}
	if mod&tea.ModAlt != 0 {
		dst = append(dst, 0x1b)
	}
	if k.Text != "" {
		return append(dst, k.Text...)
	}
	return utf8.AppendRune(dst, r)
}

func cursorKey(dst []byte, final byte, param int, appCursor bool) []byte {
	switch {
	case param != 1:
		return append(strconv.AppendInt(append(dst, "\x1b[1;"...), int64(param), 10), final)
	case appCursor:
		return append(dst, 0x1b, 'O', final)
	}
	return append(dst, 0x1b, '[', final)
}

func tilde(dst []byte, code, param int) []byte {
	dst = strconv.AppendInt(append(dst, 0x1b, '['), int64(code), 10)
	if param != 1 {
		dst = strconv.AppendInt(append(dst, ';'), int64(param), 10)
	}
	return append(dst, '~')
}

func alt(dst []byte, mod tea.KeyMod, b byte) []byte {
	if mod&tea.ModAlt != 0 {
		dst = append(dst, 0x1b)
	}
	return append(dst, b)
}
