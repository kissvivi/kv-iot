package modbusserver

import "encoding/binary"

const (
	functionReadCoils              byte = 0x01
	functionReadDiscreteInputs     byte = 0x02
	functionReadHoldingRegisters   byte = 0x03
	functionReadInputRegisters     byte = 0x04
	functionWriteSingleCoil        byte = 0x05
	functionWriteSingleRegister    byte = 0x06
	functionWriteMultipleCoils     byte = 0x0F
	functionWriteMultipleRegisters byte = 0x10
)

const (
	exceptionIllegalFunction byte = 0x01
	exceptionIllegalAddress  byte = 0x02
	exceptionIllegalValue    byte = 0x03
)

func (s *Server) handlePDU(pdu []byte) []byte {
	if len(pdu) == 0 {
		return nil
	}
	switch pdu[0] {
	case functionReadCoils:
		return s.readBits(pdu, false)
	case functionReadDiscreteInputs:
		return s.readBits(pdu, true)
	case functionReadHoldingRegisters:
		return s.readRegisters(pdu, false)
	case functionReadInputRegisters:
		return s.readRegisters(pdu, true)
	case functionWriteSingleCoil:
		return s.writeSingleCoil(pdu)
	case functionWriteSingleRegister:
		return s.writeSingleRegister(pdu)
	case functionWriteMultipleCoils:
		return s.writeMultipleCoils(pdu)
	case functionWriteMultipleRegisters:
		return s.writeMultipleRegisters(pdu)
	default:
		return exception(pdu[0], exceptionIllegalFunction)
	}
}

func (s *Server) readBits(pdu []byte, discrete bool) []byte {
	fc := pdu[0]
	if len(pdu) != 5 {
		return exception(fc, exceptionIllegalValue)
	}
	start := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	if quantity == 0 || quantity > 2000 {
		return exception(fc, exceptionIllegalValue)
	}

	var values []bool
	var ok bool
	if discrete {
		values, ok = s.data.readDiscreteInputs(start, quantity)
	} else {
		values, ok = s.data.readCoils(start, quantity)
	}
	if !ok {
		return exception(fc, exceptionIllegalAddress)
	}

	byteCount := (int(quantity) + 7) / 8
	response := make([]byte, 2+byteCount)
	response[0] = fc
	response[1] = byte(byteCount)
	for i, value := range values {
		if value {
			response[2+i/8] |= 1 << uint(i%8)
		}
	}
	return response
}

func (s *Server) readRegisters(pdu []byte, input bool) []byte {
	fc := pdu[0]
	if len(pdu) != 5 {
		return exception(fc, exceptionIllegalValue)
	}
	start := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	if quantity == 0 || quantity > 125 {
		return exception(fc, exceptionIllegalValue)
	}

	var values []uint16
	var ok bool
	if input {
		values, ok = s.data.readInputRegisters(start, quantity)
	} else {
		values, ok = s.data.readHoldingRegisters(start, quantity)
	}
	if !ok {
		return exception(fc, exceptionIllegalAddress)
	}

	response := make([]byte, 2+len(values)*2)
	response[0] = fc
	response[1] = byte(len(values) * 2)
	for i, value := range values {
		binary.BigEndian.PutUint16(response[2+i*2:], value)
	}
	return response
}

func (s *Server) writeSingleCoil(pdu []byte) []byte {
	fc := pdu[0]
	if len(pdu) != 5 {
		return exception(fc, exceptionIllegalValue)
	}
	value := binary.BigEndian.Uint16(pdu[3:5])
	if value != 0x0000 && value != 0xFF00 {
		return exception(fc, exceptionIllegalValue)
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	if !s.data.writeCoils(addr, []bool{value == 0xFF00}) {
		return exception(fc, exceptionIllegalAddress)
	}
	return append([]byte(nil), pdu...)
}

func (s *Server) writeSingleRegister(pdu []byte) []byte {
	fc := pdu[0]
	if len(pdu) != 5 {
		return exception(fc, exceptionIllegalValue)
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	value := binary.BigEndian.Uint16(pdu[3:5])
	if !s.data.writeHoldingRegisters(addr, []uint16{value}) {
		return exception(fc, exceptionIllegalAddress)
	}
	return append([]byte(nil), pdu...)
}

func (s *Server) writeMultipleCoils(pdu []byte) []byte {
	fc := pdu[0]
	if len(pdu) < 6 {
		return exception(fc, exceptionIllegalValue)
	}
	start := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	byteCount := int(pdu[5])
	if quantity == 0 || quantity > 1968 || byteCount != (int(quantity)+7)/8 || len(pdu) != 6+byteCount {
		return exception(fc, exceptionIllegalValue)
	}

	values := make([]bool, quantity)
	for i := range values {
		values[i] = pdu[6+i/8]&(1<<uint(i%8)) != 0
	}
	if !s.data.writeCoils(start, values) {
		return exception(fc, exceptionIllegalAddress)
	}
	return append([]byte(nil), pdu[:5]...)
}

func (s *Server) writeMultipleRegisters(pdu []byte) []byte {
	fc := pdu[0]
	if len(pdu) < 6 {
		return exception(fc, exceptionIllegalValue)
	}
	start := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	byteCount := int(pdu[5])
	if quantity == 0 || quantity > 123 || byteCount != int(quantity)*2 || len(pdu) != 6+byteCount {
		return exception(fc, exceptionIllegalValue)
	}

	values := make([]uint16, quantity)
	for i := range values {
		values[i] = binary.BigEndian.Uint16(pdu[6+i*2:])
	}
	if !s.data.writeHoldingRegisters(start, values) {
		return exception(fc, exceptionIllegalAddress)
	}
	return append([]byte(nil), pdu[:5]...)
}

func exception(function byte, code byte) []byte {
	return []byte{function | 0x80, code}
}
