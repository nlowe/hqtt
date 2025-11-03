package hass

import "github.com/nlowe/hqtt/mqtt"

type StateClass string

const (
	// StateClassMeasurement indicates the state represents a measurement in present time, not a historical aggregation
	// such as statistics or a prediction of the future. Examples of what should be classified StateClassMeasurement
	// are: current temperature, humidity or electric power. Examples of what should not be StateClassMeasurement:
	// Forecasted temperature for tomorrow, yesterday's energy consumption or anything else that doesn't include the
	// current measurement.
	//
	// For supported sensors, statistics of hourly min, max and average sensor readings are updated by Home Assistant
	// every 5 minutes.
	StateClassMeasurement = "measurement"

	// StateClassMeasurementAngle indicates the state represents a measurement in present time for angles measured in
	// degrees (°).
	//
	// An example of what should be classified StateClassMeasurementAngle is current wind direction
	StateClassMeasurementAngle = "measurement_angle"

	// StateClassTotal indicates the state represents a total amount that can both increase and decrease, e.g. a net
	// energy meter. Statistics of the accumulated growth or decline of the sensor's value since it was first added is
	// updated by Home Assistant every 5 minutes. This state class should not be used for sensors where the absolute
	// value is interesting instead of the accumulated growth or decline, for example remaining battery capacity or CPU
	// load; in such cases state class StateClassMeasurement should be used instead.
	StateClassTotal = "total"

	// StateClassTotalIncreasing indicates the state represents	a monotonically increasing positive total which
	// periodically restarts counting from 0, e.g. a daily amount of consumed gas, weekly water consumption or lifetime
	// energy consumption. Statistics of the accumulated growth of the sensor's value since it was first added is
	// updated by Home Assistant every 5 minutes. A decreasing value is interpreted as the start of a new meter cycle or
	// the replacement of the meter.
	StateClassTotalIncreasing = "total_increasing"
)

var (
	StateClassMarshaler mqtt.ValueMarshaler[StateClass] = func(v StateClass) ([]byte, error) {
		return mqtt.StringMarshaler(string(v))
	}
	StateClassUnmarshaler mqtt.ValueUnmarshaler[StateClass] = func(bytes []byte) (StateClass, error) {
		v, err := mqtt.StringUnmarshaler(bytes)
		return StateClass(v), err
	}
)
