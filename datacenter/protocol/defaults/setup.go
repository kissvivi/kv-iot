package defaults

import (
	"fmt"

	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/protocol/modbus"
	"kv-iot/datacenter/protocol/mqtt"
	"kv-iot/datacenter/protocol/tcpframe"
	"kv-iot/datacenter/protocol/tcpraw"
)

type driverRegistration struct {
	protocolType string
	factory      protocol.DriverFactory
	validator    protocol.ConfigValidator
}

// Register registers every protocol driver shipped with this build.
func Register(registry *protocol.Registry) error {
	if registry == nil {
		return fmt.Errorf("%w: registry is required", protocol.ErrInvalidConfig)
	}

	registrations := []driverRegistration{
		{
			protocolType: protocol.ProtocolMQTT,
			factory:      mqtt.NewDriver,
			validator:    mqtt.Validator,
		},
		{
			protocolType: protocol.ProtocolModbusTCP,
			factory:      modbus.NewDriver,
			validator:    protocol.ConfigValidatorFunc(modbus.ValidateConfig),
		},
		{
			protocolType: protocol.ProtocolTCPRaw,
			factory:      tcpraw.NewDriver,
			validator:    protocol.ConfigValidatorFunc(tcpraw.ValidateConfig),
		},
		{
			protocolType: protocol.ProtocolTCPFrame,
			factory:      tcpframe.NewDriver,
			validator:    protocol.ConfigValidatorFunc(tcpframe.ValidateConfig),
		},
	}

	for _, registration := range registrations {
		if err := registry.RegisterWithValidator(
			registration.protocolType,
			registration.factory,
			registration.validator,
		); err != nil {
			return err
		}
	}
	return nil
}

func NewRegistry() (*protocol.Registry, error) {
	registry := protocol.NewRegistry()
	if err := Register(registry); err != nil {
		return nil, err
	}
	return registry, nil
}
