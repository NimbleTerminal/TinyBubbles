package settings

import (
	tea "charm.land/bubbletea/v2"
	"testing"
)

func press(m *Model, code rune) {
	_, cmd := m.Update(tea.KeyPressMsg{Code: code})
	if cmd != nil {
		m.Update(cmd())
	}
}
func TestSettingsFlow(t *testing.T) {
	m := New()
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.CellView()
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	press(m, tea.KeyDown)
	if m.menu.SelectedIndex() != 0 {
		t.Fatal("pending selection accepted input")
	}
	m.Update(cmd())
	press(m, tea.KeyRight)
	if m.Values().Volume != 50 {
		t.Fatal("draft leaked")
	}
	m.CellView()
	press(m, tea.KeyEscape)
	if m.Values().Volume != 50 || m.focus != "" {
		t.Fatal("cancel")
	}
	press(m, tea.KeyEnter)
	press(m, tea.KeyRight)
	press(m, tea.KeyEnter)
	if m.Values().Volume != 55 {
		t.Fatal("apply volume")
	}
	press(m, tea.KeyDown)
	press(m, tea.KeyEnter)
	press(m, tea.KeyRight)
	press(m, tea.KeyEnter)
	if m.Values().Sound || m.menu.SelectedIndex() != 1 {
		t.Fatal("sound or menu focus")
	}
	press(m, tea.KeyDown)
	press(m, tea.KeyEnter)
	press(m, tea.KeyRight)
	press(m, tea.KeyEnter)
	if m.Values().Playback != 1 {
		t.Fatal("playback")
	}
	press(m, tea.KeyDown)
	press(m, tea.KeyEnter)
	press(m, tea.KeyEnter)
	if m.Values().Volume != 55 {
		t.Fatal("reset default was destructive")
	}
	press(m, tea.KeyEnter)
	press(m, tea.KeyRight)
	press(m, tea.KeyEnter)
	if m.Values() != (Values{50, true, 0}) {
		t.Fatal("reset failed", m.Values())
	}
	if m.menu.SelectedIndex() != 3 {
		t.Fatal("reset lost focus")
	}
}
func TestFocusedHelp(t *testing.T) {
	m := New()
	if got := m.Keys().ShortHelp()[0].Keys()[0]; got != "up" {
		t.Fatal(got)
	}
	press(m, tea.KeyEnter)
	if got := m.Keys().ShortHelp()[0].Keys()[0]; got != "left" {
		t.Fatal(got)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	for _, k := range m.Keys().ShortHelp() {
		if k.Enabled() {
			t.Fatal("pending help enabled")
		}
	}
	m.Update(cmd())
	if !m.Keys().ShortHelp()[0].Enabled() {
		t.Fatal("help remained disabled")
	}
}
