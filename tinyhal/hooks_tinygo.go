//go:build tinygo && !baremetal

package tinyhal

import (
	"machine"
	"unsafe"
)

func byteSlice(ptr *byte, length int) []byte {
	if ptr == nil || length <= 0 {
		return nil
	}
	return unsafe.Slice(ptr, length)
}

//export __tinygo_gpio_configure
func tinygoGPIOConfigure(pin machine.Pin, config machine.PinConfig) {
	Default().GPIOConfigure(Pin(pin), PinConfig{Mode: PinMode(config.Mode)})
}

//export __tinygo_gpio_set
func tinygoGPIOSet(pin machine.Pin, value bool) {
	Default().GPIOSet(Pin(pin), value)
}

//export __tinygo_gpio_get
func tinygoGPIOGet(pin machine.Pin) bool {
	return Default().GPIOGet(Pin(pin))
}

//export __tinygo_pwm_configure
func tinygoPWMConfigure(instance int32, frequency float64, top uint32) {
	Default().PWMConfigure(instance, frequency, top)
}

//export __tinygo_pwm_channel_configure
func tinygoPWMChannelConfigure(instance, channel int32, pin machine.Pin) {
	Default().PWMChannelConfigure(instance, channel, Pin(pin))
}

//export __tinygo_pwm_channel_set
func tinygoPWMChannelSet(instance int32, channel uint8, value uint32) {
	Default().PWMChannelSet(instance, channel, value)
}

//export __tinygo_spi_configure
func tinygoSPIConfigure(bus uint8, sck machine.Pin, sdo machine.Pin, sdi machine.Pin) {
	Default().SPIConfigure(bus, SPIConfig{
		SCK: Pin(sck),
		SDO: Pin(sdo),
		SDI: Pin(sdi),
	})
}

//export __tinygo_spi_transfer
func tinygoSPITransfer(bus uint8, w uint8) uint8 {
	return Default().SPITransfer(bus, w)
}

//export __tinygo_spi_tx
func tinygoSPITX(bus uint8, wptr *byte, wlen int, rptr *byte, rlen int) uint8 {
	Default().SPITx(bus, byteSlice(wptr, wlen), byteSlice(rptr, rlen))
	return 0
}

//export __tinygo_adc_read
func tinygoADCRead(pin machine.Pin) uint16 {
	return Default().ADCRead(Pin(pin))
}

//export __tinygo_i2c_configure
func tinygoI2CConfigure(bus uint8, scl machine.Pin, sda machine.Pin, frequency uint32) {
	Default().I2CConfigure(bus, I2CConfig{
		SCL:       Pin(scl),
		SDA:       Pin(sda),
		Frequency: frequency,
	})
}

//export __tinygo_i2c_set_baud_rate
func tinygoI2CSetBaudRate(bus uint8, br uint32) {
	Default().I2CSetBaudRate(bus, br)
}

//export __tinygo_i2c_transfer
func tinygoI2CTransfer(bus uint8, addr uint16, w *byte, wlen int, r *byte, rlen int) int {
	return Default().I2CTransfer(bus, addr, byteSlice(w, wlen), byteSlice(r, rlen))
}

//export __tinygo_uart_configure
func tinygoUARTConfigure(bus uint8, tx machine.Pin, rx machine.Pin) {
	Default().UARTConfigure(bus, UARTConfig{
		TX: Pin(tx),
		RX: Pin(rx),
	})
}

//export __tinygo_uart_read
func tinygoUARTRead(bus uint8, buf *byte, bufLen int) int {
	return Default().UARTRead(bus, byteSlice(buf, bufLen))
}

//export __tinygo_uart_write
func tinygoUARTWrite(bus uint8, buf *byte, bufLen int) int {
	return Default().UARTWrite(bus, byteSlice(buf, bufLen))
}
