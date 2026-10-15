package tinyhal_test

import (
	"image/color"
	"io"
	"testing"

	"nimbleterminal.dev/tinybubbles/tinyhal/workbench"
)

type discardDisplayer struct{ w, h int16 }

func (d *discardDisplayer) Size() (int16, int16)              { return d.w, d.h }
func (d *discardDisplayer) SetPixel(_, _ int16, _ color.RGBA) {}
func (d *discardDisplayer) Display() error                    { return nil }

func BenchmarkWorkbenchTick_TFT_Scope(b *testing.B) {
	wb := workbench.New(workbench.Config{
		Width:  240,
		Height: 135,
		Target: &discardDisplayer{w: 240, h: 135},
	})
	wb.PressMode() // SINE wave mode

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb.Tick()
	}
}

func BenchmarkWorkbenchTick_TFT_LogicAnalyzer(b *testing.B) {
	wb := workbench.New(workbench.Config{
		Width:  240,
		Height: 135,
		Target: &discardDisplayer{w: 240, h: 135},
	})
	wb.ToggleView() // LOGIC analyzer view

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb.Tick()
	}
}

func BenchmarkWorkbenchTick_OLED(b *testing.B) {
	wb := workbench.New(workbench.Config{
		Width:      128,
		Height:     64,
		Monochrome: true,
		Target:     &discardDisplayer{w: 128, h: 64},
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb.Tick()
	}
}

func BenchmarkWorkbenchVCDExport(b *testing.B) {
	wb := workbench.New(workbench.Config{
		Width:  240,
		Height: 135,
		Target: &discardDisplayer{w: 240, h: 135},
	})
	for i := 0; i < 16; i++ {
		wb.Tick()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := wb.Tracer().WriteVCD(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}
