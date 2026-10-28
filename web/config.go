package web

// Screen describes a physical display preset. Sizes are pixels.
type Screen struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Width      int16  `json:"width"`
	Height     int16  `json:"height"`
	Monochrome bool   `json:"monochrome,omitempty"`
}
type Runtime struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}
type Demo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	Runtimes    []Runtime `json:"runtimes,omitempty"`
	// TextInput forwards typed text, Ctrl+letter, named keys, and paste from
	// the focused display to the program instead of only enabled help-map
	// keys. Use it for demos that host a terminal or text editor.
	TextInput bool `json:"textInput,omitempty"`
}

// TextPart is plain introduction text with an optional link.
type TextPart struct {
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

// Config supplies page content and display sizes from Go.
type Config struct {
	Title        string     `json:"title"`
	Logo         string     `json:"logo,omitempty"`
	Introduction []TextPart `json:"introduction,omitempty"`
	Description  string     `json:"description"`
	Source       string     `json:"source"`
	Note         string     `json:"note,omitempty"`
	Screens      []Screen   `json:"screens"`
	Demos        []Demo     `json:"demos,omitempty"`
	LEDCount     int        `json:"ledCount,omitempty"`
}
type Selection struct {
	Screen  Screen  `json:"screen"`
	Demo    Demo    `json:"demo"`
	Runtime Runtime `json:"runtime"`
}
