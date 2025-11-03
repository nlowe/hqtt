package discovery

import (
	"strings"

	"github.com/nlowe/hqtt/mqtt"
)

// Constants for device fields and other fields shared by all platforms
const (
	FieldStateTopic   = "stat_t"
	FieldCommandTopic = "cmd_t"

	FieldDevice          = "dev"
	FieldOrigin          = "o"
	FieldComponents      = "cmps"
	FieldEntityCategory  = "ent_cat"
	FieldIcon            = "ic"
	FieldPicture         = "picture"
	FieldPlatform        = "p"
	FieldDefaultEntityID = "def_ent_id"
	FieldUniqueID        = "uniq_id"

	FieldPayloadOn  = "pl_on"
	FieldPayloadOff = "pl_off"

	FieldOnCommandType = "on_cmd_type"

	FieldOptimistic = "opt"

	// IDSep is the separator used to separate various parts of a device ID. It is also used as a replacement for tokens
	// that are not allowed in an ID string.
	IDSep = "__"
)

var (
	// IDSanitizer is a strings.Replacer that sanitizes a device ID for use in an MQTT Topic.
	// TODO: Are there any other tokens we need to replace?
	IDSanitizer = strings.NewReplacer(
		" ", IDSep,
		":", IDSep,
		".", IDSep,
		"!", IDSep,
		"?", IDSep,
		mqtt.TopicSeparator, IDSep,
	)
)
