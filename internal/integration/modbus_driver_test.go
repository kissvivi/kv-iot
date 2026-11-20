package integration_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
	modbusdriver "kv-iot/datacenter/protocol/modbus"
	"kv-iot/internal/testsupport/modbusserver"
)

func TestModbusDriverReadWriteThroughSimulator(t *testing.T) {
	server := modbusserver.New(modbusserver.Config{
		Coils:            8,
		HoldingRegisters: 16,
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start Modbus simulator: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close Modbus simulator: %v", err)
		}
	})

	if err := server.Data().SetCoil(2, false); err != nil {
		t.Fatalf("seed coil: %v", err)
	}
	if err := server.Data().SetHoldingRegister(0, 0x1234); err != nil {
		t.Fatalf("seed holding register: %v", err)
	}
	if err := server.Data().SetHoldingRegister(1, 0x5678); err != nil {
		t.Fatalf("seed holding register: %v", err)
	}

	host, port := splitHostPort(t, server.Addr())
	registry := protocol.NewRegistry()
	if err := registry.RegisterWithValidator(
		protocol.ProtocolModbusTCP,
		modbusdriver.NewDriver,
		protocol.ConfigValidatorFunc(modbusdriver.ValidateConfig),
	); err != nil {
		t.Fatalf("register Modbus driver: %v", err)
	}
	driverConfig := protocol.DriverConfig{
		ChannelID:    "modbus-integration",
		ProtocolType: protocol.ProtocolModbusTCP,
		Config: mustJSON(t, map[string]any{
			"host":         host,
			"port":         port,
			"unitId":       1,
			"addressBase":  0,
			"readTimeout":  "1s",
			"writeTimeout": "1s",
		}),
	}
	driver, err := registry.Create(protocol.ProtocolModbusTCP, driverConfig)
	if err != nil {
		t.Fatalf("create Modbus driver: %v", err)
	}
	t.Cleanup(func() {
		_ = driver.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, driverConfig); err != nil {
		t.Fatalf("connect Modbus driver: %v", err)
	}

	coilPoint := protocol.PointBinding{
		PointID:         201,
		DeviceID:        31,
		PointIdentifier: "door_ctl",
		Config:          mustJSON(t, map[string]any{"area": "coil", "address": 2}),
	}
	counterPoint := protocol.PointBinding{
		PointID:         202,
		DeviceID:        31,
		PointIdentifier: "counter",
		DataType:        "uint32",
		Config: mustJSON(t, map[string]any{
			"area":     "holding_register",
			"address":  0,
			"dataType": "uint32",
		}),
	}

	samples, err := driver.Read(ctx, []protocol.PointBinding{coilPoint, counterPoint})
	if err != nil {
		t.Fatalf("read Modbus points: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("Modbus sample count = %d, want 2", len(samples))
	}
	if samples[0].Value != false {
		t.Fatalf("coil value = %#v, want false", samples[0].Value)
	}
	if !reflect.DeepEqual(samples[1].Value, uint32(0x12345678)) {
		t.Fatalf("holding value = %#v, want uint32(0x12345678)", samples[1].Value)
	}

	if err := driver.Write(ctx, coilPoint, true); err != nil {
		t.Fatalf("write Modbus coil: %v", err)
	}
	value, ok := server.Data().Coil(2)
	if !ok || !value {
		t.Fatalf("simulator coil 2 = %v, %v; want true, true", value, ok)
	}

	registerPoint := protocol.PointBinding{
		PointID:         203,
		DeviceID:        31,
		PointIdentifier: "setpoint",
		DataType:        "uint16",
		Config: mustJSON(t, map[string]any{
			"area":     "holding_register",
			"address":  4,
			"dataType": "uint16",
		}),
	}
	if err := driver.Write(ctx, registerPoint, 4660); err != nil {
		t.Fatalf("write Modbus register: %v", err)
	}
	register, ok := server.Data().HoldingRegister(4)
	if !ok || register != 4660 {
		t.Fatalf("simulator holding register 4 = %d, %v; want 4660, true", register, ok)
	}

	readback, err := driver.Read(ctx, []protocol.PointBinding{registerPoint})
	if err != nil {
		t.Fatalf("read back Modbus register: %v", err)
	}
	if got := readback[0].Value; got != uint16(4660) {
		t.Fatalf("register readback = %#v, want uint16(4660)", got)
	}
}
