package hass

// EntityCategory describes the Category of an entity.
//
// An entity with a category will:
// - Not be exposed to cloud, Alexa, or Google Assistant components
// - Not be included in indirect service calls to devices or areas
//
// See https://github.com/home-assistant/core/blob/9dfee0b027d893e9f50d074550845c70ece34b4b/homeassistant/const.py#L1026-L1042
type EntityCategory string

const (
	EntityCategoryConfig     EntityCategory = "config"
	EntityCategoryDiagnostic EntityCategory = "diagnostic"
)
