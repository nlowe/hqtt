package platform

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

func setupTestSensor(t *testing.T) (*Sensor[string, map[string]any], *jsontext.Encoder, *bytes.Buffer) {
	t.Helper()

	b := &bytes.Buffer{}

	return &Sensor[string, map[string]any]{
		State: mqtt.NewValue("state", mqtt.StringMarshaler),
	}, jsontext.NewEncoder(b), b
}

func TestSensor_PlatformName(t *testing.T) {
	sut, _, _ := setupTestSensor(t)

	require.Equal(t, "sensor", sut.PlatformName())
}

func TestSensor_NoSubscription(t *testing.T) {
	sut, _, _ := setupTestSensor(t)

	require.Empty(t, sut.Subscriptions(""))
}

func TestSensor_Discovery(t *testing.T) {
	t.Parallel()

	t.Run("Required", testRequiredDiscoveryFields(t, setupTestSensor, regexpStateTopic))

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		t.Run("ExpireMeasurementsAfter", testOptional(t, setupTestSensor, func(s *Sensor[string, map[string]any]) {
			s.ExpireMeasurementsAfter = 5 * time.Minute
		}, `"exp_after"\n300`, `"exp_after"`))

		t.Run("ForceUpdate", testOptional(t, setupTestSensor, func(s *Sensor[string, map[string]any]) {
			s.ForceUpdate = true
		}, `"frc_upd"\ntrue`, `"frc_upd"`))

		t.Run("AttributesTopic", testOptional(t, setupTestSensor, func(s *Sensor[string, map[string]any]) {
			s.Attributes = NewSensorAttributeValue[map[string]any]("attr", nil)
		}, `"json_attr_t"\n"sut/attr"`, `"json_attr_t"`))

		t.Run("EnumOptions", testOptional(t, setupTestSensor, func(s *Sensor[string, map[string]any]) {
			s.EnumOptions = []string{"A", "B", "C"}
		}, `"opts"\n\["A","B","C"\]`, `"opts"`))

		t.Run("SuggestedDisplayPrecision", testOptional(t, setupTestSensor, func(s *Sensor[string, map[string]any]) {
			s.SuggestedDisplayPrecision = 2
		}, `"sug_dsp_prc"\n2`, `"sug_dsp_prc"`))

		t.Run("StateClass", testOptional(t, setupTestSensor, func(s *Sensor[string, map[string]any]) {
			s.StateClass = hass.StateClassTotalIncreasing
		}, `"stat_cla"\n"total_increasing"`, `"stat_cla"`))

		t.Run("UnitOfMeasurement", testOptional(t, setupTestSensor, func(s *Sensor[string, map[string]any]) {
			s.UnitOfMeasurement = "in"
		}, `"unit_of_meas"\n"in"`, `"unit_of_meas"`))
	})
}
