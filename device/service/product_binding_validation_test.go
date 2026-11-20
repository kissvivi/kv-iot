package service

import (
	"testing"
)

func TestValidateBindingConfig(t *testing.T) {
	tests := []struct {
		name         string
		protocolType string
		config       string
		wantValid    bool
	}{
		{
			name:         "modbus holding register",
			protocolType: "modbus_tcp",
			config:       `{"area":"holding_register","address":100,"dataType":"uint16"}`,
			wantValid:    true,
		},
		{
			name:         "modbus missing address",
			protocolType: "modbus_tcp",
			config:       `{"area":"holding_register","dataType":"uint16"}`,
			wantValid:    false,
		},
		{
			name:         "modbus invalid byte order",
			protocolType: "modbus_tcp",
			config:       `{"area":"holding_register","address":100,"dataType":"uint16","byteOrder":"middle"}`,
			wantValid:    false,
		},
		{
			name:         "mqtt uplink",
			protocolType: "mqtt",
			config:       `{"direction":"uplink","topicTemplate":"iot/{productKey}/{deviceNo}/uplink","qos":1}`,
			wantValid:    true,
		},
		{
			name:         "mqtt invalid qos",
			protocolType: "mqtt",
			config:       `{"direction":"uplink","topicTemplate":"topic","qos":3}`,
			wantValid:    false,
		},
		{
			name:         "tcp raw",
			protocolType: "tcp_raw",
			config:       `{"mode":"raw","timeoutMs":500}`,
			wantValid:    true,
		},
		{
			name:         "tcp frame missing framing",
			protocolType: "tcp_frame",
			config:       `{"requestTemplate":"AA55"}`,
			wantValid:    false,
		},
		{
			name:         "invalid json",
			protocolType: "modbus_tcp",
			config:       `[]`,
			wantValid:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validateBindingConfig(test.protocolType, test.config)
			if got := len(errs) == 0; got != test.wantValid {
				t.Fatalf("validateBindingConfig() valid = %v, want %v, errors = %v", got, test.wantValid, errs)
			}
		})
	}
}

func TestParseJSONObjectRejectsTrailingData(t *testing.T) {
	if _, err := parseJSONObject(`{"ok":true} {"extra":true}`); err == nil {
		t.Fatal("parseJSONObject() accepted trailing JSON")
	}
}

func TestValidProfileStatus(t *testing.T) {
	for _, status := range []string{"draft", "enabled", "disabled"} {
		if !validProfileStatus(status) {
			t.Fatalf("validProfileStatus(%q) = false", status)
		}
	}
	if validProfileStatus("running") {
		t.Fatal("validProfileStatus(running) = true")
	}
}
