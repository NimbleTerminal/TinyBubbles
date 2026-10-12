// Package web provides a browser harness for TinyGo display applications.
package web

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	"github.com/NimbleMarkets/go-booba/actions/bubblehelp"
)

// ActionOptions includes disabled bindings in the browser's action metadata.
func ActionOptions() bubblehelp.Options {
	return bubblehelp.Options{IncludeDisabled: true}
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
