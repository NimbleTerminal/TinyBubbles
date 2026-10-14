//go:build !tinygo || baremetal

package workbench

import "nimbleterminal.dev/tinybubbles/tinyhal"

type mcuBus struct {
	hal *tinyhal.HAL
}

func newWorkbenchHAL() *tinyhal.HAL {
	return tinyhal.New()
}

func newMCUBus(h *tinyhal.HAL) mcuBus {
	return mcuBus{hal: h}
}

func (b *mcuBus) configurePinsAndBuses(width, height int16, mono bool) {
	h := b.hal
	h.GPIOConfigure(PinButtonMode, tinyhal.PinConfig{Mode: tinyhal.PinInputPullup})
	h.GPIOConfigure(PinButtonLED, tinyhal.PinConfig{Mode: tinyhal.PinInputPulldown})
	h.GPIOConfigure(PinButtonBurst, tinyhal.PinConfig{Mode: tinyhal.PinInputPulldown})

	for _, p := range []tinyhal.Pin{PinLEDStatus, PinLEDPWM, PinLEDActivity, PinSPICS, PinTFTDC, PinTFTReset, PinTFTCS, PinTFTLite} {
		h.GPIOConfigure(p, tinyhal.PinConfig{Mode: tinyhal.PinOutput})
	}
	h.GPIOSet(PinSPICS, true)
	h.GPIOSet(PinTFTCS, true)
	h.GPIOSet(PinTFTReset, true)
	h.GPIOSet(PinTFTLite, true)

	h.SPIConfigure(0, tinyhal.SPIConfig{
		Frequency: 20_000_000,
		SCK:       PinSPISCK,
		SDO:       PinSPISDO,
		SDI:       tinyhal.NoPin,
	})
	h.I2CConfigure(0, tinyhal.I2CConfig{
		Frequency: 400_000,
		SDA:       4,
		SCL:       5,
	})
	h.UARTConfigure(0, tinyhal.UARTConfig{
		TX: 1,
		RX: 2,
	})

	h.PWMConfigure(0, 1000, 4095)
	h.PWMChannelConfigure(0, 0, PinLEDPWM)

	if mono {
		_ = h.I2C(0).Tx(I2COLEDAddr, []byte{0x00, 0xAE, 0x20, 0x00, 0xA6, 0xAF}, nil)
	} else {
		h.GPIOSet(PinTFTCS, false)
		b.sendTFTCmd(0x11)       // SLPOUT
		b.sendTFTCmd(0x3A, 0x55) // COLMOD RGB565
		b.sendTFTCmd(0x29)       // DISPON
		h.GPIOSet(PinTFTCS, true)
	}
	_, _ = width, height
}

func (b *mcuBus) sendTFTCmd(cmd byte, data ...byte) {
	b.hal.GPIOSet(PinTFTDC, false)
	b.hal.SPITx(0, []byte{cmd}, nil)
	b.hal.GPIOSet(PinTFTDC, true)
	if len(data) > 0 {
		b.hal.SPITx(0, data, nil)
	}
}

func (b *mcuBus) pinGet(pin tinyhal.Pin) bool {
	return b.hal.GPIOGet(pin)
}

func (b *mcuBus) pinSet(pin tinyhal.Pin, high bool) {
	b.hal.GPIOSet(pin, high)
}

func (b *mcuBus) adcRead(pin tinyhal.Pin) uint16 {
	return b.hal.ADCRead(pin)
}

func (b *mcuBus) pwmSet(value uint32) {
	b.hal.PWMChannelSet(0, 0, value)
}

func (b *mcuBus) spiTx(w, r []byte) {
	b.hal.SPITx(0, w, r)
}

func (b *mcuBus) i2cTx(addr uint16, w, r []byte) error {
	return b.hal.I2C(0).Tx(addr, w, r)
}

func (b *mcuBus) uartWrite(data []byte) {
	b.hal.UARTWrite(0, data)
}

func (b *mcuBus) flushDisplay(f *frameCanvas) {
	if f.mono {
		pages := byte(max(int16(1), f.height/8) - 1)
		_ = b.hal.I2C(0).Tx(I2COLEDAddr, []byte{
			0x00,
			0x21, 0, byte(f.width - 1),
			0x22, 0, pages,
		}, nil)
		f.gddram[0] = 0x40
		_ = b.hal.I2C(0).Tx(I2COLEDAddr, f.gddram, nil)
		return
	}
	wEnd := uint16(f.width - 1)
	hEnd := uint16(f.height - 1)
	b.hal.GPIOSet(PinTFTCS, false)
	b.sendTFTCmd(0x2A, 0, 0, byte(wEnd>>8), byte(wEnd))
	b.sendTFTCmd(0x2B, 0, 0, byte(hEnd>>8), byte(hEnd))
	b.sendTFTCmd(0x2C, f.rgb565...)
	b.hal.GPIOSet(PinTFTCS, true)
}
