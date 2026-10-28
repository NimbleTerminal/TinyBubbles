package tinyghostty

import (
	"bytes"
	"testing"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
)

func model(t *testing.T, cols, rows int) (*Model, *bytes.Buffer) {
	t.Helper()
	m := New(cols, rows)
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	t.Cleanup(m.Close)
	in := &bytes.Buffer{}
	m.Input = in
	return m, in
}

func TestOutputAndView(t *testing.T) {
	m, _ := model(t, 10, 2)
	m.Update(OutputMsg("hello\r\n\x1b[31mred"))
	if got := m.Text(); got != "hello\nred" {
		t.Fatalf("view %q", got)
	}
	buf := m.CellView()
	if buf.At(0, 1).Rune != 'r' || buf.At(0, 1).Style == bubbles.Normal {
		t.Fatalf("styled cell %+v", buf.At(0, 1))
	}
	m.Update(tea.WindowSizeMsg{Width: 4, Height: 3})
	if w, h := m.CellView().Size(); w != 4 || h != 3 {
		t.Fatalf("resized buffer %dx%d", w, h)
	}
}

func TestSender(t *testing.T) {
	var got []tea.Msg
	w := Sender(func(msg tea.Msg) { got = append(got, msg) })
	p := []byte("ab")
	w.Write(p)
	p[0] = 'x'
	if len(got) != 1 || string(got[0].(OutputMsg)) != "ab" {
		t.Fatalf("sent %q", got)
	}
}

func TestKeysFollowCursorMode(t *testing.T) {
	m, in := model(t, 10, 2)
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m.Update(OutputMsg("\x1b[?1h"))
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m.Update(tea.KeyPressMsg{Code: 'x', Text: "x", Mod: tea.ModAlt})
	if got := in.String(); got != "\x1b[A\x1bOA\x03\x1bx" {
		t.Fatalf("input %q", got)
	}
}

func TestAppendKey(t *testing.T) {
	for _, c := range []struct {
		key  tea.Key
		want string
	}{
		{tea.Key{Code: 'a', Text: "a"}, "a"},
		{tea.Key{Code: '界'}, "界"},
		{tea.Key{Code: tea.KeyEnter}, "\r"},
		{tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}, "\x1b[Z"},
		{tea.Key{Code: tea.KeyBackspace}, "\x7f"},
		{tea.Key{Code: tea.KeyEscape}, "\x1b"},
		{tea.Key{Code: tea.KeyRight, Mod: tea.ModCtrl}, "\x1b[1;5C"},
		{tea.Key{Code: tea.KeyDelete}, "\x1b[3~"},
		{tea.Key{Code: tea.KeyPgDown, Mod: tea.ModShift}, "\x1b[6;2~"},
		{tea.Key{Code: tea.KeyF1}, "\x1bOP"},
		{tea.Key{Code: tea.KeyF12}, "\x1b[24~"},
		{tea.Key{Code: '[', Mod: tea.ModCtrl}, "\x1b"},
		{tea.Key{Code: tea.KeyF40}, ""},
	} {
		if got := string(AppendKey(nil, c.key, false)); got != c.want {
			t.Errorf("%v: %q, want %q", c.key, got, c.want)
		}
	}
}

func TestPaste(t *testing.T) {
	m, in := model(t, 10, 2)
	m.Update(tea.PasteMsg{Content: "a\nb"})
	m.Update(OutputMsg("\x1b[?2004h"))
	m.Update(tea.PasteMsg{Content: "c\x1b[201~d"})
	if got := in.String(); got != "a\rb\x1b[200~cd\x1b[201~" {
		t.Fatalf("paste %q", got)
	}
}

func TestReplies(t *testing.T) {
	m, in := model(t, 10, 4)
	m.Write([]byte("\x1b[3;2H\x1b[6n"))
	if got := in.String(); got != "\x1b[3;2R" {
		t.Fatalf("reply %q", got)
	}
}
