package tinycharts

import (
	"image/color"
	"testing"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
)

type testDisplayer struct {
	w, h   int16
	pixels int
}

func (d *testDisplayer) Size() (int16, int16) { return d.w, d.h }
func (d *testDisplayer) Display() error       { return nil }
func (d *testDisplayer) SetPixel(_, _ int16, c color.RGBA) {
	if c.R != 0 || c.G != 0 || c.B != 0 {
		d.pixels++
	}
}

func TestRunesAndBrailleGrid(t *testing.T) {
	if LowerBlock(0) != ' ' || LowerBlock(4) != RuneLowerBlockFour || LowerBlock(8) != RuneFullBlock {
		t.Fatalf("unexpected LowerBlock mapping")
	}
	if LeftBlock(0) != ' ' || LeftBlock(4) != RuneLeftBlockFour || LeftBlock(8) != RuneFullBlock {
		t.Fatalf("unexpected LeftBlock mapping")
	}
	if QuadrantBlock(true, false, false, true) != '\u259A' {
		t.Fatalf("unexpected QuadrantBlock(UL, LR)")
	}

	g := NewBrailleGrid(4, 2)
	dw, dh := g.DotSize()
	if dw != 8 || dh != 8 {
		t.Fatalf("expected 8x8 dot grid, got %dx%d", dw, dh)
	}
	g.DrawLine(0, 7, 7, 0)
	buf := bubbles.NewBuffer(4, 2)
	g.Draw(buf, layout.Rect{Width: 4, Height: 2}, bubbles.Selected)

	nonEmpty := 0
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			if r := buf.At(x, y).Rune; r >= RuneBrailleBlank && r <= 0x28FF {
				nonEmpty++
			}
		}
	}
	if nonEmpty == 0 {
		t.Fatalf("expected Braille line to populate cells")
	}
}

func TestSparklineAndBarChart(t *testing.T) {
	sp := NewSparkline(8, 3, WithSparklineMax(100), WithSparklineStyle(bubbles.Accent))
	sp.PushAll([]float64{10, 25, 50, 75, 100})
	sp.Resize(6, 3)
	v := sp.View()
	if v.At(5, 0).Rune != RuneFullBlock {
		t.Fatalf("expected newest max sample to fill top cell, got %U", v.At(5, 0).Rune)
	}

	bc := NewBarChart(16, 3, WithBarMax(100), WithBarHorizontal(true))
	bc.SetBars([]BarData{
		{Label: "ADC", Value: 50, Style: bubbles.Selected},
		{Label: "PWM", Value: 100, Style: bubbles.Accent},
	})
	bv := bc.View()
	lit := 0
	for y := 0; y < 3; y++ {
		for x := 0; x < 16; x++ {
			if bv.At(x, y).Rune != 0 && bv.At(x, y).Rune != ' ' {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Fatalf("expected horizontal BarChart to draw cells")
	}

	bc.Horizontal = false
	bv = bc.View()
	if bv.At(0, 2).Rune == 0 {
		t.Fatalf("expected vertical BarChart to draw bottom row")
	}
}

func TestStreamlineModesAndLogicChart(t *testing.T) {
	for _, mode := range []StreamMode{StreamModeBraille, StreamModeArc, StreamModeBox} {
		sl := NewStreamline(10, 4, WithStreamRange(0, 4095), WithStreamMode(mode))
		sl.PushAll([]float64{0, 1024, 2048, 4095, 3000, 512, 2048})
		v := sl.View()
		lit := 0
		for y := 0; y < 4; y++ {
			for x := 0; x < 10; x++ {
				if r := v.At(x, y).Rune; r != 0 && r != ' ' {
					lit++
				}
			}
		}
		if lit == 0 {
			t.Fatalf("expected Streamline mode %d to render non-blank runes", mode)
		}
	}

	lc := NewLogicChart(16, 8, []LogicChannel{
		{Label: "G13", Kind: LogicBit, Style: bubbles.Accent},
		{Label: "PWM", Kind: LogicPWM, Style: bubbles.Selected},
		{Label: "ACT", Kind: LogicBit, Style: bubbles.Normal},
		{Label: "BUS", Kind: LogicBus, Style: bubbles.Accent},
	})
	lc.PushStep(1, 128, 0, 0)
	lc.PushStep(0, 255, 1, 1)
	lc.PushStep(1, 64, 0, 0)
	lv := lc.View()

	disp := &testDisplayer{w: 16 * 6, h: 8 * 8}
	PaintBuffer(disp, 0, 0, lv, bubbles.DefaultFont(), bubbles.DefaultTheme(false))
	if disp.pixels == 0 {
		t.Fatalf("expected PaintBuffer with DefaultFont to rasterize LogicChart Unicode glyphs")
	}
}
