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
}

// Config supplies page content and display sizes from Go.
type Config struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	Note        string   `json:"note,omitempty"`
	Screens     []Screen `json:"screens"`
	Demos       []Demo   `json:"demos,omitempty"`
	LEDCount    int      `json:"ledCount,omitempty"`
}
type Selection struct {
	Screen  Screen  `json:"screen"`
	Demo    Demo    `json:"demo"`
	Runtime Runtime `json:"runtime"`
}
