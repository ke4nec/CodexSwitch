package codexswitch

import (
	"bufio"
	"strconv"
	"strings"
)

type parsedConfig struct {
	Model                string
	ReviewModel          string
	ModelReasoningEffort string
	BaseURL              string
	Values               map[string]string
}

func parseConfigTOML(raw string) parsedConfig {
	cfg := parsedConfig{Values: map[string]string{}}
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := stripTOMLComment(scanner.Text())
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, value, ok := splitTOMLAssignment(line)
		if !ok {
			continue
		}
		fullKey := key
		if section != "" {
			fullKey = section + "." + key
		}
		parsedValue := parseTOMLString(value)
		cfg.Values[fullKey] = parsedValue

		switch {
		case fullKey == "model":
			cfg.Model = parsedValue
		case fullKey == "review_model":
			cfg.ReviewModel = parsedValue
		case fullKey == "model_reasoning_effort":
			cfg.ModelReasoningEffort = parsedValue
		case fullKey == "base_url" && cfg.BaseURL == "":
			cfg.BaseURL = parsedValue
		case strings.HasSuffix(fullKey, ".base_url") && cfg.BaseURL == "":
			cfg.BaseURL = parsedValue
		}
	}
	return cfg
}

func hasActiveBaseURLLine(raw string) bool {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(stripTOMLComment(scanner.Text()))
		if line == "" {
			continue
		}

		key, _, ok := splitTOMLAssignment(line)
		if !ok {
			continue
		}
		if key == "base_url" || strings.HasSuffix(key, ".base_url") {
			return true
		}
	}
	return false
}

func stripTOMLComment(line string) string {
	var builder strings.Builder
	inQuotes := false
	escaped := false
	for _, r := range line {
		switch {
		case escaped:
			builder.WriteRune(r)
			escaped = false
		case r == '\\':
			builder.WriteRune(r)
			escaped = inQuotes
		case r == '"':
			builder.WriteRune(r)
			inQuotes = !inQuotes
		case r == '#' && !inQuotes:
			return builder.String()
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func splitTOMLAssignment(line string) (string, string, bool) {
	inQuotes := false
	escaped := false
	for i, r := range line {
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = inQuotes
		case r == '"':
			inQuotes = !inQuotes
		case r == '=' && !inQuotes:
			key := strings.TrimSpace(line[:i])
			value := strings.TrimSpace(line[i+1:])
			if key == "" || value == "" {
				return "", "", false
			}
			return key, value, true
		}
	}
	return "", "", false
}

func parseTOMLString(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			return unquoted
		}
	}
	return value
}

func tomlQuote(value string) string {
	return strconv.Quote(value)
}
