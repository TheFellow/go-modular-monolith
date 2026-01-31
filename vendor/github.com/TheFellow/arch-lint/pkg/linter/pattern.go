package linter

import (
	"fmt"
	"regexp"
	"strings"
)

func matchPattern(pattern, path string) (map[string]string, bool) {
	regexPattern := escapePattern(pattern)

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, false
	}

	// Match the path against the regex
	match := re.FindStringSubmatch(path)
	if match == nil {
		return nil, false
	}

	// Extract named groups into a map
	vars := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i > 0 && name != "" {
			vars[name] = match[i]
		}
	}
	return vars, true
}

func escapePattern(pattern string) string {
	// Split the pattern into segments
	segments := strings.Split(pattern, "/")
	for i, segment := range segments {
		// Handle variables
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			// Convert {var} to (?P<var>[^/]+)
			segment = fmt.Sprintf("(?P<%s>[^/]+)", segment[1:len(segment)-1])
		}

		// Handle single-level wildcards
		if segment == "*" {
			// Convert * to [^/]+
			segment = "[^/]+"
		}

		// Handle multi-level wildcards
		if segment == "**" {
			// Convert ** to .*
			segment = ".*"
		}

		// Update the segment
		segments[i] = segment
	}

	// Join the segments back together
	regexPattern := strings.Join(segments, "/")
	// Special case for /** at the end of the pattern
	if strings.HasSuffix(regexPattern, "/.*") {
		// If the pattern ends with a wildcard, allow empty string at the end
		regexPattern = strings.TrimSuffix(regexPattern, "/.*") + "/?.*"
	}
	regexPattern = "^" + regexPattern + "$"
	return regexPattern
}

func replaceVariables(pattern string, vars map[string]string) string {
	segments := strings.Split(pattern, "/")
	for i, segment := range segments {
		for key := range vars {
			// Handle negated variables by treating them as normal variables in the regex
			negatedPlaceholder := fmt.Sprintf("{!%s}", key)
			if segment == negatedPlaceholder {
				segment = fmt.Sprintf("{%s}", key)
			}
		}
		segments[i] = segment
	}
	return strings.Join(segments, "/")
}

func exceptRegex(pattern, path string, vars map[string]string) bool {
	// Replace variables in the pattern
	regexPattern := escapePattern(replaceVariables(pattern, vars))

	// Compile the regex
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	// Match the path against the regex
	match := re.FindStringSubmatch(path)
	if match == nil {
		return false
	}

	// Extract named groups into a map
	capturedVars := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i > 0 && name != "" {
			capturedVars[name] = match[i]
		}
	}

	// Validate variables (both positive and negated)
	for key, value := range vars {
		negatedPlaceholder := fmt.Sprintf("{!%s}", key)
		positivePlaceholder := fmt.Sprintf("{%s}", key)

		if strings.Contains(pattern, negatedPlaceholder) {
			// For negated variables, ensure the captured value does not match the forbidden value
			if capturedVars[key] == value {
				return false // Negated variable matches the forbidden value
			}
		} else if strings.Contains(pattern, positivePlaceholder) {
			// For positive variables, ensure the captured value matches the expected value
			if capturedVars[key] != value {
				return false // Positive variable does not match the expected value
			}
		}
	}

	return true
}
