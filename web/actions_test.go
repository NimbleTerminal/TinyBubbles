package web

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
)

type keys []key.Binding

func (k keys) ShortHelp() []key.Binding  { return k[:min(5, len(k))] }
func (k keys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp(), k[min(5, len(k)):]} }

type actionModel struct{ k keys }

func (*actionModel) Init() tea.Cmd                         { return nil }
func (m *actionModel) Update(tea.Msg) (tea.Model, tea.Cmd) { m.k[0].SetEnabled(false); return m, nil }
func (*actionModel) View() tea.View                        { return tea.NewView("") }
func musicKeys() keys {
	names := []string{"up", "down", "left", "right", "enter", "esc", "space", "+", "-", "n", "p"}
	labels := []string{"Up", "Down", "Left", "Right", "Select", "Back", "Play/Pause", "Vol +", "Vol -", "Next", "Previous"}
	var k keys
	for i, name := range names {
		k = append(k, key.NewBinding(key.WithKeys(name), key.WithHelp(name, labels[i])))
	}
	return k
}
func decode(t *testing.T, cmd tea.Cmd) actions.Snapshot {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				return decode(t, c)
			}
		}
	}
	raw, ok := msg.(tea.RawMsg)
	if !ok {
		t.Fatalf("unexpected message %T", msg)
	}
	_, encoded, ok := strings.Cut(raw.Msg.(string), ":msg=")
	if !ok {
		t.Fatal("missing status")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(encoded, "\x1b\\"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > actions.MaxStatusMessageBytes {
		t.Fatalf("payload has %d bytes", len(data))
	}
	_, payload, _ := strings.Cut(string(data), actions.MessageMarker)
	var snapshot actions.Snapshot
	if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatal(err)
	}
	return snapshot
}
func TestCompactActionsFitStatusAndFollowModel(t *testing.T) {
	m := &actionModel{k: musicKeys()}
	wrapped, err := WrapActions(m, "tinypod", func(m tea.Model) help.KeyMap { return m.(*actionModel).k }, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, cmd := wrapped.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	snapshot := decode(t, cmd)
	if len(snapshot.Actions) != 11 {
		t.Fatalf("lost actions: %d", len(snapshot.Actions))
	}
	for _, a := range snapshot.Actions {
		if len(a.ID) > 6 {
			t.Fatalf("long ID: %s", a.ID)
		}
	}
	_, cmd = wrapped.Update(struct{}{})
	next := decode(t, cmd)
	if next.Actions[0].Enabled || next.Revision <= snapshot.Revision {
		t.Fatal("stale action state")
	}
}
func TestActionValidation(t *testing.T) {
	for _, k := range []keys{
		{key.NewBinding(key.WithKeys("a"), key.WithHelp("a", strings.Repeat("x", 2000)))},
		{key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "One")), key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "Two"))},
	} {
		_, err := WrapActions(&actionModel{k: k}, "test", func(m tea.Model) help.KeyMap { return m.(*actionModel).k }, nil)
		if err == nil {
			t.Fatal("expected invalid payload or ID collision error")
		}
	}
	m := &actionModel{k: keys{key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "Find")), key.NewBinding(key.WithKeys("é"), key.WithHelp("é", "Accent"))}}
	if _, err := WrapActions(m, "test", func(m tea.Model) help.KeyMap { return m.(*actionModel).k }, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := WrapActions(m, "test", func(tea.Model) help.KeyMap { return nil }, nil); err == nil {
		t.Fatal("nil key map accepted")
	}
}
