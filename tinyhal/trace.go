package tinyhal

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"sync"
)

// TraceKind identifies the peripheral type of a recorded bus/pin trace sample.
type TraceKind string

const (
	TraceGPIO TraceKind = "gpio"
	TracePWM  TraceKind = "pwm"
	TraceSPI  TraceKind = "spi"
	TraceI2C  TraceKind = "i2c"
	TraceUART TraceKind = "uart"
)

// TraceSample records a timestamped pin transition or bus transaction.
type TraceSample struct {
	TimeUS uint64
	Kind   TraceKind
	Pin    Pin
	Bus    uint8
	Addr   uint16
	High   bool
	PWM    uint8
	TX     []byte
	RX     []byte
}

// DefaultMaxTraceSamples bounds in-memory trace history so long-running browser
// and workbench sessions do not grow without limit.
const DefaultMaxTraceSamples = 4096

// TraceRecorder captures GPIO, PWM, SPI, I2C, and UART transitions and exports
// IEEE 1364 Value Change Dump (.vcd) waveforms for tools such as Surfer or GTKWave.
type TraceRecorder struct {
	mu         sync.Mutex
	timeUS     uint64
	maxSamples int
	samples    []TraceSample
}

// NewTraceRecorder creates an empty bus and pin trace recorder bounded to DefaultMaxTraceSamples.
func NewTraceRecorder() *TraceRecorder {
	return &TraceRecorder{
		maxSamples: DefaultMaxTraceSamples,
		samples:    make([]TraceSample, 0, 256),
	}
}

// Reset clears all recorded samples and resets the trace timestamp to 0.
func (t *TraceRecorder) Reset() {
	t.mu.Lock()
	t.timeUS = 0
	t.samples = t.samples[:0]
	t.mu.Unlock()
}

// Advance increments the simulated trace clock by deltaUS microseconds.
func (t *TraceRecorder) Advance(deltaUS uint64) {
	t.mu.Lock()
	t.timeUS += deltaUS
	t.mu.Unlock()
}

// Len returns the number of recorded trace samples without allocating.
func (t *TraceRecorder) Len() int {
	t.mu.Lock()
	n := len(t.samples)
	t.mu.Unlock()
	return n
}

// Samples returns a copy of all recorded trace samples.
func (t *TraceRecorder) Samples() []TraceSample {
	t.mu.Lock()
	out := make([]TraceSample, len(t.samples))
	copy(out, t.samples)
	t.mu.Unlock()
	return out
}

func (t *TraceRecorder) appendSampleLocked(s TraceSample, deltaUS uint64) {
	if t.maxSamples > 0 && len(t.samples) >= t.maxSamples {
		copy(t.samples, t.samples[1:])
		t.samples[len(t.samples)-1] = s
	} else {
		t.samples = append(t.samples, s)
	}
	t.timeUS += deltaUS
}

func (t *TraceRecorder) recordGPIO(pin Pin, high bool) {
	t.mu.Lock()
	t.appendSampleLocked(TraceSample{
		TimeUS: t.timeUS,
		Kind:   TraceGPIO,
		Pin:    pin,
		High:   high,
	}, 1)
	t.mu.Unlock()
}

func (t *TraceRecorder) recordPWM(pin Pin, duty uint8) {
	t.mu.Lock()
	t.appendSampleLocked(TraceSample{
		TimeUS: t.timeUS,
		Kind:   TracePWM,
		Pin:    pin,
		PWM:    duty,
	}, 1)
	t.mu.Unlock()
}

func (t *TraceRecorder) recordSPI(bus uint8, w, r []byte) {
	tx := append([]byte(nil), w...)
	rx := append([]byte(nil), r...)
	t.mu.Lock()
	n := max(1, max(len(tx), len(rx)))
	t.appendSampleLocked(TraceSample{
		TimeUS: t.timeUS,
		Kind:   TraceSPI,
		Bus:    bus,
		TX:     tx,
		RX:     rx,
	}, uint64(n))
	t.mu.Unlock()
}

func (t *TraceRecorder) recordI2C(bus uint8, addr uint16, w, r []byte) {
	tx := append([]byte(nil), w...)
	rx := append([]byte(nil), r...)
	t.mu.Lock()
	n := max(1, len(tx)+len(rx))
	t.appendSampleLocked(TraceSample{
		TimeUS: t.timeUS,
		Kind:   TraceI2C,
		Bus:    bus,
		Addr:   addr,
		TX:     tx,
		RX:     rx,
	}, uint64(n))
	t.mu.Unlock()
}

func (t *TraceRecorder) recordUART(bus uint8, data []byte) {
	tx := append([]byte(nil), data...)
	t.mu.Lock()
	t.appendSampleLocked(TraceSample{
		TimeUS: t.timeUS,
		Kind:   TraceUART,
		Bus:    bus,
		TX:     tx,
	}, uint64(max(1, len(tx))))
	t.mu.Unlock()
}

type vcdSignal struct {
	name  string
	width int
	id    string
}

func vcdID(index int) string {
	// Printable ASCII VCD identifiers start at '!' (33) through '~' (126).
	const span = 94
	var buf [4]byte
	pos := len(buf)
	for {
		pos--
		buf[pos] = byte(33 + (index % span))
		index = index/span - 1
		if index < 0 {
			break
		}
	}
	return string(buf[pos:])
}

// WriteVCD writes the recorded trace in IEEE 1364 Value Change Dump (.vcd) format.
func (t *TraceRecorder) WriteVCD(w io.Writer) error {
	samples := t.Samples()

	sigMap := make(map[string]*vcdSignal)
	addSig := func(name string, width int) {
		if _, exists := sigMap[name]; !exists {
			sigMap[name] = &vcdSignal{name: name, width: width}
		}
	}
	for _, s := range samples {
		switch s.Kind {
		case TraceGPIO:
			addSig(fmt.Sprintf("gpio_%d", s.Pin), 1)
		case TracePWM:
			addSig(fmt.Sprintf("pwm_%d", s.Pin), 8)
		case TraceSPI:
			addSig(fmt.Sprintf("spi%d_mosi", s.Bus), 8)
			addSig(fmt.Sprintf("spi%d_miso", s.Bus), 8)
		case TraceI2C:
			addSig(fmt.Sprintf("i2c%d_addr", s.Bus), 8)
			addSig(fmt.Sprintf("i2c%d_data", s.Bus), 8)
		case TraceUART:
			addSig(fmt.Sprintf("uart%d_tx", s.Bus), 8)
		}
	}

	names := make([]string, 0, len(sigMap))
	for name := range sigMap {
		names = append(names, name)
	}
	sort.Strings(names)
	for i, name := range names {
		sigMap[name].id = vcdID(i)
	}

	var out bytes.Buffer
	out.WriteString("$version tinyhal VCD recorder $end\n")
	out.WriteString("$timescale 1us $end\n")
	out.WriteString("$scope module tinyhal $end\n")
	for _, name := range names {
		sig := sigMap[name]
		fmt.Fprintf(&out, "$var wire %d %s %s $end\n", sig.width, sig.id, sig.name)
	}
	out.WriteString("$upscope $end\n")
	out.WriteString("$enddefinitions $end\n")

	// Emit initial values at #0.
	out.WriteString("#0\n")
	for _, name := range names {
		sig := sigMap[name]
		if sig.width == 1 {
			fmt.Fprintf(&out, "0%s\n", sig.id)
		} else {
			fmt.Fprintf(&out, "b00000000 %s\n", sig.id)
		}
	}

	lastTime := uint64(0)
	hasTime := false
	emitTime := func(ts uint64) {
		if !hasTime || ts != lastTime {
			fmt.Fprintf(&out, "#%d\n", ts)
			lastTime = ts
			hasTime = true
		}
	}

	for _, s := range samples {
		switch s.Kind {
		case TraceGPIO:
			emitTime(s.TimeUS)
			sig := sigMap[fmt.Sprintf("gpio_%d", s.Pin)]
			bit := '0'
			if s.High {
				bit = '1'
			}
			fmt.Fprintf(&out, "%c%s\n", bit, sig.id)
		case TracePWM:
			emitTime(s.TimeUS)
			sig := sigMap[fmt.Sprintf("pwm_%d", s.Pin)]
			fmt.Fprintf(&out, "b%08b %s\n", s.PWM, sig.id)
		case TraceSPI:
			mosiSig := sigMap[fmt.Sprintf("spi%d_mosi", s.Bus)]
			misoSig := sigMap[fmt.Sprintf("spi%d_miso", s.Bus)]
			n := max(len(s.TX), len(s.RX))
			for i := 0; i < n; i++ {
				emitTime(s.TimeUS + uint64(i))
				var txByte, rxByte byte
				if i < len(s.TX) {
					txByte = s.TX[i]
				}
				if i < len(s.RX) {
					rxByte = s.RX[i]
				}
				fmt.Fprintf(&out, "b%08b %s\n", txByte, mosiSig.id)
				fmt.Fprintf(&out, "b%08b %s\n", rxByte, misoSig.id)
			}
		case TraceI2C:
			addrSig := sigMap[fmt.Sprintf("i2c%d_addr", s.Bus)]
			dataSig := sigMap[fmt.Sprintf("i2c%d_data", s.Bus)]
			emitTime(s.TimeUS)
			fmt.Fprintf(&out, "b%08b %s\n", byte(s.Addr), addrSig.id)
			step := uint64(0)
			for _, b := range s.TX {
				emitTime(s.TimeUS + step)
				fmt.Fprintf(&out, "b%08b %s\n", b, dataSig.id)
				step++
			}
			for _, b := range s.RX {
				emitTime(s.TimeUS + step)
				fmt.Fprintf(&out, "b%08b %s\n", b, dataSig.id)
				step++
			}
		case TraceUART:
			txSig := sigMap[fmt.Sprintf("uart%d_tx", s.Bus)]
			for i, b := range s.TX {
				emitTime(s.TimeUS + uint64(i))
				fmt.Fprintf(&out, "b%08b %s\n", b, txSig.id)
			}
		}
	}

	_, err := w.Write(out.Bytes())
	return err
}

// VCDString renders the recorded trace as an IEEE 1364 .vcd string.
func (t *TraceRecorder) VCDString() string {
	var buf bytes.Buffer
	_ = t.WriteVCD(&buf)
	return buf.String()
}
