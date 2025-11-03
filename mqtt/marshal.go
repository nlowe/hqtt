package mqtt

import (
	"encoding/json"
	"strconv"
)

// ValueMarshaler is a function that can convert values of type T to a byte slice for writing to an MQTT Topic.
type ValueMarshaler[T any] func(v T) ([]byte, error)

// ValueUnmarshaler is a function that can convert the byte slice payload from an MQTT Message to values of type T.
type ValueUnmarshaler[T any] func([]byte) (T, error)

var (
	StringMarshaler ValueMarshaler[string] = func(v string) ([]byte, error) {
		return []byte(v), nil
	}

	StringUnmarshaler ValueUnmarshaler[string] = func(bytes []byte) (string, error) {
		return string(bytes), nil
	}

	UintMarshaler ValueMarshaler[uint] = func(v uint) ([]byte, error) {
		return []byte(strconv.Itoa(int(v))), nil
	}
	UintUnmarshaler ValueUnmarshaler[uint] = func(bytes []byte) (uint, error) {
		v, err := strconv.ParseUint(string(bytes), 10, 64)
		return uint(v), err
	}
)

// JsonValueMarshaler returns a ValueMarshaler for type T implemented by marshaling the value to Json.
func JsonValueMarshaler[T any]() ValueMarshaler[T] {
	return func(v T) ([]byte, error) {
		return json.Marshal(v)
	}
}

// JsonValueUnmarshaler returns a ValueUnmarshaler for type T implemented by un-marshaling the payload from json.
func JsonValueUnmarshaler[T any]() ValueUnmarshaler[T] {
	return func(bytes []byte) (T, error) {
		var v T

		return v, json.Unmarshal(bytes, &v)
	}
}
