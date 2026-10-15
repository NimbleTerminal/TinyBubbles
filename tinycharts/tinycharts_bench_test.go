package tinycharts

import (
	"testing"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
)

func BenchmarkBrailleGridDrawLine(b *testing.B) {
	g := NewBrailleGrid(40, 8)
	dst := bubbles.NewBuffer(40, 8)
	rect := layout.Rect{Width: 40, Height: 8}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Clear()
		g.DrawLine(0, 31, 79, i%32)
		g.Draw(dst, rect, bubbles.Accent)
	}
}

func BenchmarkStreamlineBraille(b *testing.B) {
	sl := NewStreamline(40, 8, WithStreamRange(0, 4095), WithStreamMode(StreamModeBraille))
	for i := 0; i < 80; i++ {
		sl.Push(float64((i * 51) % 4096))
	}
	dst := bubbles.NewBuffer(40, 8)
	rect := layout.Rect{Width: 40, Height: 8}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Push(float64((i * 97) % 4096))
		sl.Draw(dst, rect)
	}
}

func BenchmarkStreamlineArc(b *testing.B) {
	sl := NewStreamline(40, 8, WithStreamRange(0, 4095), WithStreamMode(StreamModeArc))
	for i := 0; i < 40; i++ {
		sl.Push(float64((i * 103) % 4096))
	}
	dst := bubbles.NewBuffer(40, 8)
	rect := layout.Rect{Width: 40, Height: 8}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Push(float64((i * 97) % 4096))
		sl.Draw(dst, rect)
	}
}

func BenchmarkSparkline(b *testing.B) {
	sp := NewSparkline(40, 6, WithSparklineMax(100), WithSparklineAutoMax(false))
	for i := 0; i < 40; i++ {
		sp.Push(float64((i * 7) % 101))
	}
	dst := bubbles.NewBuffer(40, 6)
	rect := layout.Rect{Width: 40, Height: 6}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sp.Push(float64((i * 13) % 101))
		sp.Draw(dst, rect)
	}
}

func BenchmarkBarChartHorizontal(b *testing.B) {
	bc := NewBarChart(40, 4, WithBarMax(100), WithBarHorizontal(true))
	bc.SetBars([]BarData{
		{Label: "ADC", Value: 72, Style: bubbles.Accent},
		{Label: "PWM", Value: 48, Style: bubbles.Selected},
		{Label: "SPI", Value: 91, Style: bubbles.Normal},
	})
	dst := bubbles.NewBuffer(40, 4)
	rect := layout.Rect{Width: 40, Height: 4}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bc.Draw(dst, rect)
	}
}

func BenchmarkLogicChart(b *testing.B) {
	lc := NewLogicChart(40, 8, []LogicChannel{
		{Label: "G13", Kind: LogicBit, Style: bubbles.Accent},
		{Label: "PWM", Kind: LogicPWM, Style: bubbles.Selected},
		{Label: "ACT", Kind: LogicBit, Style: bubbles.Normal},
		{Label: "BUS", Kind: LogicBus, Style: bubbles.Accent},
	})
	for i := 0; i < 40; i++ {
		lc.PushStep(uint16(i&1), uint16((i*17)&0xff), uint16((i/4)&1), uint16(i%2))
	}
	dst := bubbles.NewBuffer(40, 8)
	rect := layout.Rect{Width: 40, Height: 8}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lc.PushStep(uint16(i&1), uint16((i*23)&0xff), uint16((i/8)&1), uint16(i&3))
		lc.Draw(dst, rect)
	}
}
