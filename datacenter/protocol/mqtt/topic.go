package mqtt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Topic templates use {name}, while payload templates in existing documents
// commonly use ${name}. Both forms resolve through the same variable map.
var templateVariablePattern = regexp.MustCompile(`\$?\{([^{}]+)\}`)

func validateTopicFilter(filter string) error {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return fmt.Errorf("%w: topic filter is required", ErrInvalidTopic)
	}
	levels := strings.Split(filter, "/")
	for index, level := range levels {
		if strings.Contains(level, "#") {
			if level != "#" || index != len(levels)-1 {
				return fmt.Errorf("%w: # must occupy the final level", ErrInvalidTopic)
			}
		}
		if strings.Contains(level, "+") && level != "+" {
			return fmt.Errorf("%w: + must occupy a complete level", ErrInvalidTopic)
		}
	}
	return nil
}

func validatePublishTopic(topic string) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("%w: publish topic is required", ErrInvalidTopic)
	}
	if strings.ContainsAny(topic, "+#") {
		return fmt.Errorf("%w: publish topic must not contain wildcards", ErrInvalidTopic)
	}
	return nil
}

func matchTopicFilter(filter, topic string) bool {
	filterLevels := strings.Split(filter, "/")
	topicLevels := strings.Split(topic, "/")
	for index, filterLevel := range filterLevels {
		if filterLevel == "#" {
			return index == len(filterLevels)-1
		}
		if index >= len(topicLevels) {
			return false
		}
		if filterLevel == "+" {
			continue
		}
		if filterLevel != topicLevels[index] {
			return false
		}
	}
	return len(filterLevels) == len(topicLevels)
}

// topicTemplateToFilter converts a concrete topic template into an MQTT
// subscription filter. A level containing one or more variables becomes "+".
func topicTemplateToFilter(template string) (string, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return "", fmt.Errorf("%w: topicTemplate is required", ErrInvalidTopic)
	}
	levels := strings.Split(template, "/")
	for index, level := range levels {
		if templateVariablePattern.MatchString(level) {
			levels[index] = "+"
		}
	}
	filter := strings.Join(levels, "/")
	if err := validateTopicFilter(filter); err != nil {
		return "", err
	}
	return filter, nil
}

// matchTopicTemplate returns captured template variables. Both ${variable}
// levels and regular MQTT wildcards are supported.
func matchTopicTemplate(template, topic string) (map[string]string, bool) {
	template = strings.TrimSpace(template)
	topic = strings.TrimSpace(topic)
	if template == "" {
		return nil, false
	}

	variables := make(map[string]string)
	var expression strings.Builder
	expression.WriteString("^")
	levels := strings.Split(template, "/")
	for index, level := range levels {
		if index > 0 {
			expression.WriteString("/")
		}
		if level == "#" {
			expression.WriteString(".*")
			continue
		}
		if level == "+" {
			expression.WriteString(`[^/]+`)
			continue
		}
		matches := templateVariablePattern.FindAllStringSubmatchIndex(level, -1)
		if len(matches) == 0 {
			expression.WriteString(regexp.QuoteMeta(level))
			continue
		}
		offset := 0
		for _, match := range matches {
			expression.WriteString(regexp.QuoteMeta(level[offset:match[0]]))
			name := strings.TrimSpace(level[match[2]:match[3]])
			expression.WriteString(`([^/]+)`)
			variables[strconv.Itoa(len(variables))] = name
			offset = match[1]
		}
		expression.WriteString(regexp.QuoteMeta(level[offset:]))
	}
	expression.WriteString("$")

	compiled, err := regexp.Compile(expression.String())
	if err != nil {
		return nil, false
	}
	matches := compiled.FindStringSubmatch(topic)
	if matches == nil {
		return nil, false
	}
	captured := make(map[string]string, len(variables))
	for index, name := range variables {
		position, err := strconv.Atoi(index)
		if err != nil || position+1 >= len(matches) {
			continue
		}
		captured[name] = matches[position+1]
	}
	return captured, true
}

func renderString(template string, variables map[string]any) (string, error) {
	if match := templateVariablePattern.FindStringSubmatch(template); len(match) > 0 &&
		match[0] == template {
		value, exists := variables[strings.TrimSpace(match[1])]
		if !exists {
			return "", fmt.Errorf("%w: %s", ErrTemplateVariable, match[1])
		}
		return valueToString(value), nil
	}

	var renderErr error
	result := templateVariablePattern.ReplaceAllStringFunc(template, func(placeholder string) string {
		if renderErr != nil {
			return placeholder
		}
		match := templateVariablePattern.FindStringSubmatch(placeholder)
		name := strings.TrimSpace(match[1])
		value, exists := variables[name]
		if !exists {
			renderErr = fmt.Errorf("%w: %s", ErrTemplateVariable, name)
			return placeholder
		}
		return valueToString(value)
	})
	if renderErr != nil {
		return "", renderErr
	}
	if strings.Contains(result, "${") {
		return "", fmt.Errorf("%w: unresolved placeholder in %q", ErrTemplateVariable, template)
	}
	return result, nil
}

func renderTemplateValue(value any, variables map[string]any) (any, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		match := templateVariablePattern.FindStringSubmatch(typed)
		if len(match) > 0 && match[0] == typed {
			variable, exists := variables[strings.TrimSpace(match[1])]
			if !exists {
				return nil, fmt.Errorf("%w: %s", ErrTemplateVariable, match[1])
			}
			return variable, nil
		}
		return renderString(typed, variables)
	case map[string]any:
		rendered := make(map[string]any, len(typed))
		for key, item := range typed {
			value, err := renderTemplateValue(item, variables)
			if err != nil {
				return nil, err
			}
			rendered[key] = value
		}
		return rendered, nil
	case []any:
		rendered := make([]any, 0, len(typed))
		for _, item := range typed {
			value, err := renderTemplateValue(item, variables)
			if err != nil {
				return nil, err
			}
			rendered = append(rendered, value)
		}
		return rendered, nil
	default:
		return value, nil
	}
}

func valueToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}
