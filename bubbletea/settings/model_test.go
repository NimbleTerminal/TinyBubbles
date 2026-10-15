package settings

import (
	"image/color"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/tinyhal"
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

func TestSmallMenuLayout(t *testing.T) {
	m := New()
	for _, size := range [][2]int{{21, 8}, {4, 2}, {4, 1}, {0, 0}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		b := m.CellView()
		w, h := b.Size()
		if w > 0 && h > 0 && b.At(0, 0).Rune != 'S' {
			t.Fatal("header lost")
		}
		if h > 1 && b.At(0, h-1).Rune != 'C' {
			t.Fatal("footer lost")
		}
	}
}

type featherSPIST7789 struct {
	hal                  *tinyhal.HAL
	cs, dc, rst, bl      tinyhal.Pin
	width, height        int16
	colOffset, rowOffset int16
}

func (d *featherSPIST7789) cmd(c byte, data ...byte) {
	d.hal.GPIOSet(d.dc, false)
	d.hal.SPITx(0, []byte{c}, nil)
	d.hal.GPIOSet(d.dc, true)
	if len(data) > 0 {
		d.hal.SPITx(0, data, nil)
	}
}

func (d *featherSPIST7789) Size() (int16, int16) { return d.width, d.height }

func (d *featherSPIST7789) SetPixel(x, y int16, c color.RGBA) {
	_ = d.FillRectangle(x, y, 1, 1, c)
}

func (d *featherSPIST7789) FillRectangle(x, y, width, height int16, c color.RGBA) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	x0 := uint16(x + d.colOffset)
	x1 := uint16(x + width - 1 + d.colOffset)
	y0 := uint16(y + d.rowOffset)
	y1 := uint16(y + height - 1 + d.rowOffset)
	rgb565 := (uint16(c.R&0xF8) << 8) | (uint16(c.G&0xFC) << 3) | (uint16(c.B) >> 3)
	hi, lo := byte(rgb565>>8), byte(rgb565)
	buf := make([]byte, int(width)*int(height)*2)
	for i := 0; i < len(buf); i += 2 {
		buf[i] = hi
		buf[i+1] = lo
	}
	d.hal.GPIOSet(d.cs, false)
	d.cmd(0x2A, byte(x0>>8), byte(x0), byte(x1>>8), byte(x1))
	d.cmd(0x2B, byte(y0>>8), byte(y0), byte(y1>>8), byte(y1))
	d.cmd(0x2C, buf...)
	d.hal.GPIOSet(d.cs, true)
	return nil
}

func (d *featherSPIST7789) Display() error { return nil }

func TestFeatherThreeButtonSettingsTinyHAL(t *testing.T) {
	h := tinyhal.New()
	const (
		tftPower     tinyhal.Pin = 7
		tftClock     tinyhal.Pin = 36
		tftMOSI      tinyhal.Pin = 35
		tftCS        tinyhal.Pin = 42
		tftDC        tinyhal.Pin = 40
		tftReset     tinyhal.Pin = 41
		tftBacklight tinyhal.Pin = 45
		pinD0        tinyhal.Pin = 0
		pinD1        tinyhal.Pin = 1
		pinD2        tinyhal.Pin = 2
	)

	h.GPIOConfigure(tftPower, tinyhal.PinConfig{Mode: tinyhal.PinOutput})
	h.GPIOSet(tftPower, true)

	tft := tinyhal.AttachST7789(h, tinyhal.ST7789Config{
		ID:           "feather-settings-tft",
		Bus:          0,
		Width:        240,
		Height:       135,
		RowOffset:    40,
		ColumnOffset: 53,
		CS:           tftCS,
		DC:           tftDC,
		Reset:        tftReset,
		Backlight:    tftBacklight,
	})

	h.SPIConfigure(0, tinyhal.SPIConfig{Frequency: 20_000_000, SCK: tftClock, SDO: tftMOSI, SDI: tinyhal.NoPin})
	display := &featherSPIST7789{
		hal:       h,
		cs:        tftCS,
		dc:        tftDC,
		rst:       tftReset,
		bl:        tftBacklight,
		width:     240,
		height:    135,
		colOffset: 40,
		rowOffset: 53,
	}
	for _, p := range []tinyhal.Pin{tftCS, tftDC, tftReset, tftBacklight} {
		h.GPIOConfigure(p, tinyhal.PinConfig{Mode: tinyhal.PinOutput})
	}
	h.GPIOSet(tftCS, true)
	h.GPIOSet(tftReset, true)
	h.GPIOSet(tftBacklight, true)
	h.GPIOSet(tftCS, false)
	display.cmd(0x11)       // SLPOUT
	display.cmd(0x3A, 0x55) // COLMOD 16bpp
	display.cmd(0x36, 0x68) // MADCTL MV|MX|BGR
	display.cmd(0x29)       // DISPON
	h.GPIOSet(tftCS, true)

	h.GPIOConfigure(pinD0, tinyhal.PinConfig{Mode: tinyhal.PinInputPullup})
	h.GPIOConfigure(pinD1, tinyhal.PinConfig{Mode: tinyhal.PinInputPulldown})
	h.GPIOConfigure(pinD2, tinyhal.PinConfig{Mode: tinyhal.PinInputPulldown})
	tinyhal.AttachButton(h, tinyhal.ButtonConfig{ID: "d0", Pin: pinD0, ActiveLow: true})
	tinyhal.AttachButton(h, tinyhal.ButtonConfig{ID: "d1", Pin: pinD1, ActiveLow: false})
	tinyhal.AttachButton(h, tinyhal.ButtonConfig{ID: "d2", Pin: pinD2, ActiveLow: false})

	pins := [3]tinyhal.Pin{pinD0, pinD1, pinD2}
	boardPressed := func(index int) bool {
		v := h.GPIOGet(pins[index])
		if index == 0 {
			return !v
		}
		return v
	}

	buttons := FeatherButtons()
	poller := bubbles.ButtonPoller{Buttons: buttons[:]}

	m := New()
	dm, err := displaytea.NewModel(m, display, bubbles.DefaultFont(), bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatalf("displaytea.NewModel failed: %v", err)
	}
	defer dm.Close()

	cols, rows := dm.Size()
	dm.Update(tea.WindowSizeMsg{Width: cols, Height: rows})
	_ = dm.View()

	now := time.Now()
	stepPoll := func() {
		for {
			msg := poller.Poll(now, boardPressed)
			if msg == nil {
				break
			}
			_, cmd := dm.Update(msg)
			for cmd != nil {
				nextMsg := cmd()
				_, cmd = dm.Update(nextMsg)
			}
			_ = dm.View()
		}
	}

	// Short tap helper: press, debounce 30ms, release, debounce 30ms.
	shortTap := func(id string) {
		_ = h.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: id, Pressed: true})
		stepPoll()
		now = now.Add(30 * time.Millisecond)
		stepPoll()
		_ = h.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: id, Pressed: false})
		stepPoll()
		now = now.Add(30 * time.Millisecond)
		stepPoll()
	}

	// Long press helper: press, debounce 30ms, hold 550ms, release, debounce 30ms.
	longPress := func(id string) {
		_ = h.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: id, Pressed: true})
		stepPoll()
		now = now.Add(30 * time.Millisecond)
		stepPoll()
		now = now.Add(550 * time.Millisecond)
		stepPoll()
		_ = h.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: id, Pressed: false})
		stepPoll()
		now = now.Add(30 * time.Millisecond)
		stepPoll()
	}

	// 1. Short-tap D2 to enter Volume (50%), hold D1 to auto-repeat up to 65%, then long-press D2 to cancel.
	shortTap("d2")
	if m.focus != "volume" {
		t.Fatalf("expected focus=volume after D2 short tap, got %q", m.focus)
	}
	_ = h.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d1", Pressed: true})
	stepPoll()
	now = now.Add(30 * time.Millisecond)
	stepPoll() // initial press: 50 -> 55
	now = now.Add(400 * time.Millisecond)
	stepPoll() // first repeat: 55 -> 60
	now = now.Add(100 * time.Millisecond)
	stepPoll() // second repeat: 60 -> 65
	_ = h.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d1", Pressed: false})
	stepPoll()
	now = now.Add(30 * time.Millisecond)
	stepPoll()

	longPress("d2")
	if m.focus != "" || m.Values().Volume != 50 {
		t.Fatalf("expected long-press D2 to cancel volume edit, got focus=%q values=%+v", m.focus, m.Values())
	}

	// 2. Short-tap D2 to enter Volume, tap D1 (+5 -> 55%), short-tap D2 to apply.
	shortTap("d2")
	shortTap("d1")
	shortTap("d2")
	if m.focus != "" || m.Values().Volume != 55 {
		t.Fatalf("expected applied volume=55, got focus=%q values=%+v", m.focus, m.Values())
	}

	// 3. Tap D1 to move down to Sound, short-tap D2 to enter, tap D0 to select Off, short-tap D2 to apply.
	shortTap("d1")
	shortTap("d2")
	shortTap("d0")
	shortTap("d2")
	if m.focus != "" || m.Values().Sound {
		t.Fatalf("expected Sound=false after D0/D2, got focus=%q values=%+v", m.focus, m.Values())
	}

	// 4. Tap D1 twice to reach "Reset defaults", short-tap D2 to open dialog, tap D1 to select "Reset", short-tap D2 to confirm.
	shortTap("d1")
	shortTap("d1")
	shortTap("d2")
	if m.focus != "reset" {
		t.Fatalf("expected focus=reset, got %q", m.focus)
	}
	shortTap("d1")
	shortTap("d2")
	if m.focus != "" || m.Values() != (Values{50, true, 0}) {
		t.Fatalf("expected reset defaults via D0/D1/D2, got focus=%q values=%+v", m.focus, m.Values())
	}
	if tft.Frames() < 5 {
		t.Fatalf("expected ST7789 frames rendered over tinyhal SPI0, got %d", tft.Frames())
	}
}
