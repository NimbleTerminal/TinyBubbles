// Package web provides a browser harness for TinyGo display applications.
package web

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	"github.com/NimbleMarkets/go-booba/actions/bubblehelp"
)

// ActionOptions uses compact IDs so ordinary key maps fit in OSC 7501.
func ActionOptions() bubblehelp.Options {
	return bubblehelp.Options{IncludeDisabled: true, Customize: shortID}
}

func shortID(_ key.Binding, action *actions.Action) error {
	if len(action.Keys) == 0 {
		return fmt.Errorf("action has no keys")
	}
	gesture := action.Keys[0]
	id := strings.Join(gesture.Modifiers, "+")
	if id != "" {
		id += "+"
	}
	for _, r := range gesture.Key {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_.+-", r) {
			id += string(r)
		} else {
			id += "_" + strconv.FormatInt(int64(r), 16) + "_"
		}
	}
	action.ID = id
	return nil
}

// WrapActions reads the current key map after each update. Errors reach onError.
func WrapActions(model tea.Model, context string, keys func(tea.Model) help.KeyMap, onError func(error)) (tea.Model, error) {
	if keys == nil {
		return nil, fmt.Errorf("web: missing help map callback")
	}
	validate := func(m tea.Model) error {
		km := keys(m)
		if km == nil {
			return fmt.Errorf("web: nil help map")
		}
		// Booba validates the complete OSC payload, including its summary.
		_, _, err := bubblehelp.Snapshot(km, context, context, 1, ActionOptions())
		return err
	}
	if model == nil {
		return nil, fmt.Errorf("web: nil model")
	}
	if err := validate(model); err != nil {
		return nil, fmt.Errorf("web actions: %w", err)
	}
	if onError == nil {
		onError = func(err error) { fmt.Println("web actions:", err) }
	}
	return bubblehelp.NewModel(model, func(m tea.Model) bubblehelp.Description {
		return bubblehelp.Description{Context: context, Keys: keys(m), Status: actions.ProgramStatus{Summary: context}}
	}, bubblehelp.ModelOptions{Instance: context, Adapter: ActionOptions(), OnError: onError})
}
