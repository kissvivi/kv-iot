package modbusserver

import (
	"fmt"
	"sync"
)

const defaultAddressSpace = 65536

// Data stores the address spaces exposed by a test Modbus server.
type Data struct {
	mu sync.RWMutex

	coils           []bool
	discreteInputs  []bool
	holdingRegister []uint16
	inputRegister   []uint16
}

func newData(cfg Config) *Data {
	return &Data{
		coils:           make([]bool, addressSpace(cfg.Coils)),
		discreteInputs:  make([]bool, addressSpace(cfg.DiscreteInputs)),
		holdingRegister: make([]uint16, addressSpace(cfg.HoldingRegisters)),
		inputRegister:   make([]uint16, addressSpace(cfg.InputRegisters)),
	}
}

func addressSpace(value int) int {
	if value < 0 {
		return 0
	}
	if value == 0 {
		return defaultAddressSpace
	}
	return value
}

// SetCoil updates a coil value.
func (d *Data) SetCoil(addr uint16, value bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if int(addr) >= len(d.coils) {
		return fmt.Errorf("modbusserver: coil address %d out of range", addr)
	}
	d.coils[addr] = value
	return nil
}

// Coil returns a coil value.
func (d *Data) Coil(addr uint16) (bool, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if int(addr) >= len(d.coils) {
		return false, false
	}
	return d.coils[addr], true
}

// SetDiscreteInput updates a discrete input value.
func (d *Data) SetDiscreteInput(addr uint16, value bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if int(addr) >= len(d.discreteInputs) {
		return fmt.Errorf("modbusserver: discrete input address %d out of range", addr)
	}
	d.discreteInputs[addr] = value
	return nil
}

// DiscreteInput returns a discrete input value.
func (d *Data) DiscreteInput(addr uint16) (bool, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if int(addr) >= len(d.discreteInputs) {
		return false, false
	}
	return d.discreteInputs[addr], true
}

// SetHoldingRegister updates a holding register value.
func (d *Data) SetHoldingRegister(addr uint16, value uint16) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if int(addr) >= len(d.holdingRegister) {
		return fmt.Errorf("modbusserver: holding register address %d out of range", addr)
	}
	d.holdingRegister[addr] = value
	return nil
}

// HoldingRegister returns a holding register value.
func (d *Data) HoldingRegister(addr uint16) (uint16, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if int(addr) >= len(d.holdingRegister) {
		return 0, false
	}
	return d.holdingRegister[addr], true
}

// SetInputRegister updates an input register value.
func (d *Data) SetInputRegister(addr uint16, value uint16) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if int(addr) >= len(d.inputRegister) {
		return fmt.Errorf("modbusserver: input register address %d out of range", addr)
	}
	d.inputRegister[addr] = value
	return nil
}

// InputRegister returns an input register value.
func (d *Data) InputRegister(addr uint16) (uint16, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if int(addr) >= len(d.inputRegister) {
		return 0, false
	}
	return d.inputRegister[addr], true
}

func (d *Data) readCoils(start, quantity uint16) ([]bool, bool) {
	return d.readBits(start, quantity, d.coils)
}

func (d *Data) readDiscreteInputs(start, quantity uint16) ([]bool, bool) {
	return d.readBits(start, quantity, d.discreteInputs)
}

func (d *Data) readBits(start, quantity uint16, values []bool) ([]bool, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !withinRange(start, quantity, len(values)) {
		return nil, false
	}
	result := make([]bool, quantity)
	copy(result, values[start:uint32(start)+uint32(quantity)])
	return result, true
}

func (d *Data) readHoldingRegisters(start, quantity uint16) ([]uint16, bool) {
	return d.readRegisters(start, quantity, d.holdingRegister)
}

func (d *Data) readInputRegisters(start, quantity uint16) ([]uint16, bool) {
	return d.readRegisters(start, quantity, d.inputRegister)
}

func (d *Data) readRegisters(start, quantity uint16, values []uint16) ([]uint16, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !withinRange(start, quantity, len(values)) {
		return nil, false
	}
	result := make([]uint16, quantity)
	copy(result, values[start:uint32(start)+uint32(quantity)])
	return result, true
}

func (d *Data) writeCoils(start uint16, values []bool) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !withinRange(start, uint16(len(values)), len(d.coils)) {
		return false
	}
	copy(d.coils[start:uint32(start)+uint32(len(values))], values)
	return true
}

func (d *Data) writeHoldingRegisters(start uint16, values []uint16) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !withinRange(start, uint16(len(values)), len(d.holdingRegister)) {
		return false
	}
	copy(d.holdingRegister[start:uint32(start)+uint32(len(values))], values)
	return true
}

func withinRange(start, quantity uint16, size int) bool {
	return quantity > 0 && uint32(start)+uint32(quantity) <= uint32(size)
}
