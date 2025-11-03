package mqtt

import "strings"

const TopicSeparator = "/"

// TrimTopic trims TopicSeparator from the start and end of the specified topic.
func TrimTopic(topic string) string {
	return strings.Trim(topic, TopicSeparator)
}

// JoinTopic joins non-empty component parts with TopicSeparator, trimming each part as it is appended.
func JoinTopic(parts ...string) string {
	var result strings.Builder

	for i, part := range parts {
		if part == "" || part == TopicSeparator {
			continue
		}
		result.WriteString(TrimTopic(part))

		if i != len(parts)-1 {
			result.WriteString(TopicSeparator)
		}
	}

	return result.String()
}
