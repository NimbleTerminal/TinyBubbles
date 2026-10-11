package value

import (
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/value"
	"testing"
)

func TestKeysAndCapturedEvents(t *testing.T) {
	m, err := New(cells.Config{Min: 0, Max: 10, Step: 5, Value: 5})
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.Keys().Increase.Enabled() || !m.Keys().Decrease.Enabled() {
		t.Fatal("upper-bound help")
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Adjust(-1)
	if cmd().(AppliedMsg).Value != 10 {
		t.Fatal("apply did not capture value")
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd().(CancelledMsg).Value != 10 || m.Value() != 10 {
		t.Fatal("cancel did not restore")
	}
	m.SetValue(0)
	if m.Keys().Decrease.Enabled() || !m.Keys().Increase.Enabled() {
		t.Fatal("lower-bound help")
	}
	m.KeyMap.Increase.SetEnabled(false)
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.Value() != 0 || m.Keys().Increase.Enabled() {
		t.Fatal("disabled custom key accepted")
	}
	if _, err := New(cells.Config{}); err == nil {
		t.Fatal("invalid configuration accepted")
	}
}
