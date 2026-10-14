//go:build tinygo && !baremetal

package workbench

import (
	"machine"

	"nimbleterminal.dev/tinybubbles/tinyhal"
)

type mcuBus struct {
	hal  *tinyhal.HAL
	i2c0 machine.I2C
}

func newWorkbenchHAL() *tinyhal.HAL {
	h := tinyhal.Default()
	h.Reset()
	return h
}

func newMCUBus(h *tinyhal.HAL) mcuBus {
	return mcuBus{
		hal:  h,
		i2c0: machine.I2C{Bus: 0},
	}
}

func (b *mcuBus) configurePinsAndBuses(width, height int16, mono bool) {
	machine.Pin(PinButtonMode).Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	machine.Pin(PinButtonLED).Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	machine.Pin(PinButtonBurst).Configure(machine.PinConfig{Mode: machine.PinInputPulldown})

	for _, p := range []tinyhal.Pin{PinLEDStatus, PinLEDPWM, PinLEDActivity, PinSPICS, PinTFTDC, PinTFTReset, PinTFTCS, PinTFTLite} {
		machine.Pin(p).Configure(machine.PinConfig{Mode: machine.PinOutput})
	}
	machine.Pin(PinSPICS).High()
	machine.Pin(PinTFTCS).High()
	machine.Pin(PinTFTReset).High()
	machine.Pin(PinTFTLite).High()

	machine.InitADC()
	adc := machine.ADC{Pin: machine.Pin(PinADCPot)}
	adc.Configure(machine.ADCConfig{})

	_ = machine.SPI0.Configure(machine.SPIConfig{
		Frequency: 20_000_000,
		SCK:       machine.Pin(PinSPISCK),
		SDO:       machine.Pin(PinSPISDO),
		SDI:       machine.NoPin,
	})
	_ = b.i2c0.Configure(machine.I2CConfig{
		Frequency: 400_000,
		SDA:       machine.Pin(4),
		SCL:       machine.Pin(5),
	})
	machine.UART0.Configure(machine.UARTConfig{
		TX: machine.Pin(1),
		RX: machine.Pin(2),
	})

	// Configure PWM timer 0 channel 0 on PinLEDPWM with a 12-bit top (4095).
	b.hal.PWMConfigure(0, 1000, 4095)
	b.hal.PWMChannelConfigure(0, 0, PinLEDPWM)

	if mono {
		_ = b.i2c0.Tx(I2COLEDAddr, []byte{0x00, 0xAE, 0x20, 0x00, 0xA6, 0xAF}, nil)
	} else {
		machine.Pin(PinTFTCS).Low()
		b.sendTFTCmd(0x11)       // SLPOUT
		b.sendTFTCmd(0x3A, 0x55) // COLMOD RGB565
		b.sendTFTCmd(0x29)       // DISPON
		machine.Pin(PinTFTCS).High()
	}
	_, _ = width, height
}

func (b *mcuBus) sendTFTCmd(cmd byte, data ...byte) {
	machine.Pin(PinTFTDC).Low()
	_ = machine.SPI0.Tx([]byte{cmd}, nil)
	machine.Pin(PinTFTDC).High()
	if len(data) > 0 {
		_ = machine.SPI0.Tx(data, nil)
	}
}

func (b *mcuBus) pinGet(pin tinyhal.Pin) bool {
	return machine.Pin(pin).Get()
}

func (b *mcuBus) pinSet(pin tinyhal.Pin, high bool) {
	machine.Pin(pin).Set(high)
}

func (b *mcuBus) adcRead(pin tinyhal.Pin) uint16 {
	return (machine.ADC{Pin: machine.Pin(pin)}).Get()
}

func (b *mcuBus) pwmSet(value uint32) {
	b.hal.PWMChannelSet(0, 0, value)
}

func (b *mcuBus) spiTx(w, r []byte) {
	_ = machine.SPI0.Tx(w, r)
}

func (b *mcuBus) i2cTx(addr uint16, w, r []byte) error {
	return b.i2c0.Tx(addr, w, r)
}

func (b *mcuBus) uartWrite(data []byte) {
	_, _ = machine.UART0.Write(data)
}

func (b *mcuBus) flushDisplay(f *frameCanvas) {
	if f.mono {
		pages := byte(max(int16(1), f.height/8) - 1)
		_ = b.i2c0.Tx(I2COLEDAddr, []byte{
			0x00,
			0x21, 0, byte(f.width - 1),
			0x22, 0, pages,
		}, nil)
		f.gddram[0] = 0x40
		_ = b.i2c0.Tx(I2COLEDAddr, f.gddram, nil)
		return
	}
	wEnd := uint16(f.width - 1)
	hEnd := uint16(f.height - 1)
	machine.Pin(PinTFTCS).Low()
	b.sendTFTCmd(0x2A, 0, 0, byte(wEnd>>8), byte(wEnd))
	b.sendTFTCmd(0x2B, 0, 0, byte(hEnd>>8), byte(hEnd))
	b.sendTFTCmd(0x2C, f.rgb565...)
	machine.Pin(PinTFTCS).High()
}
