package hass

type UpdateDeviceClass string

const (
	UpdateDeviceClassDefault  UpdateDeviceClass = UpdateDeviceClassGeneric
	UpdateDeviceClassGeneric  UpdateDeviceClass = ""
	UpdateDeviceClassFirmware UpdateDeviceClass = "firmware"
)
