package workbench

import (
	"image/color"
	"testing"
)

type testDisplay struct {
	w, h    int16
	flushes int
	lit     int
}

func (d *testDisplay) Size() (int16, int16) { return d.w, d.h }

func (d *testDisplay) SetPixel(_, _ int16, c color.RGBA) {
	if c.R != 0 || c.G != 0 || c.B != 0 {
		d.lit++
	}
}

func (d *testDisplay) Display() error {
	d.flushes++
	return nil
}

type testLEDs struct {
	colors []color.RGBA
}

func (l *testLEDs) WriteColors(colors []color.RGBA) error {
	l.colors = append(l.colors[:0], colors...)
	return nil
}

func TestWorkbenchColorAndMonochrome(t *testing.T) {
	disp := &testDisplay{w: 240, h: 135}
	leds := &testLEDs{}
	wb := New(Config{
		Width:      240,
		Height:     135,
		Monochrome: false,
		Target:     disp,
		LEDs:       leds,
	})

	if !wb.LEDStatusOn() {
		t.Fatalf("expected initial GPIO13 status LED on")
	}
	if wb.ADC() != 2048 || wb.PWMPercent() == 0 {
		t.Fatalf("expected initial ADC=2048 and non-zero PWM, got adc=%d pwm=%d%%", wb.ADC(), wb.PWMPercent())
	}
	if disp.flushes == 0 || disp.lit == 0 {
		t.Fatalf("expected ST7789 SPI frame flush to target display")
	}
	if len(leds.colors) != 3 {
		t.Fatalf("expected 3 LED colors written, got %d", len(leds.colors))
	}

	// Toggle GPIO13 LED via simulated button D1.
	wb.ToggleLED()
	if wb.LEDStatusOn() {
		t.Fatalf("expected D1 press to toggle GPIO13 status LED off")
	}

	// Adjust ADC and I2C sensor temperature.
	wb.AdjustADC(1024)
	if wb.ADC() != 3072 {
		t.Fatalf("expected ADC=3072, got %d", wb.ADC())
	}
	wb.AdjustTemp(5)
	if wb.TempC() != 29 {
		t.Fatalf("expected TempC=29, got %d", wb.TempC())
	}

	// Trigger SPI + I2C + UART burst via button D2.
	prevReg := wb.SPIRegister()
	wb.TriggerBurst()
	if wb.SPIRegister() == prevReg {
		t.Fatalf("expected D2 burst to advance SPI register value, stayed 0x%02x", prevReg)
	}

	// Cycle waveform modes via active-low button D0 and advance ticks.
	wb.PressMode()
	if wb.Mode() != ModeSine {
		t.Fatalf("expected ModeSine after D0 press, got %v", wb.Mode())
	}
	wb.Tick()
	wb.PressMode()
	if wb.Mode() != ModeSaw {
		t.Fatalf("expected ModeSaw after second D0 press, got %v", wb.Mode())
	}
	wb.Tick()

	// Toggle to VCD Logic Analyzer view and verify VCD waveform output.
	if wb.ViewMode() != ViewScope {
		t.Fatalf("expected initial ViewScope, got %v", wb.ViewMode())
	}
	wb.ToggleView()
	if wb.ViewMode() != ViewLogic {
		t.Fatalf("expected ViewLogic after ToggleView, got %v", wb.ViewMode())
	}
	wb.Tick()
	if len(wb.Tracer().Samples()) == 0 || wb.VCD() == "" {
		t.Fatalf("expected recorded VCD trace samples in workbench")
	}

	// Verify monochrome SSD1306 I2C workbench mode (128x64) in both Scope and Logic views.
	oledDisp := &testDisplay{w: 128, h: 64}
	oledWB := New(Config{
		Width:      128,
		Height:     64,
		Monochrome: true,
		Target:     oledDisp,
	})
	oledWB.Tick()
	oledWB.ToggleView()
	oledWB.Tick()
	if oledDisp.flushes < 3 || oledDisp.lit == 0 {
		t.Fatalf("expected SSD1306 I2C frame flushes, got flushes=%d lit=%d", oledDisp.flushes, oledDisp.lit)
	}
}
