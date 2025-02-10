package handler

import (
	"fmt"
	"strings"
)

// processInput validates and converts the input to a string for hashing
func processInput(input interface{}) (string, error) {
	switch v := input.(type) {
	case string:
		if v == "" {
			return "", fmt.Errorf("empty input string")
		}
		return v, nil
	case []interface{}:
		return processNumberArray(v)
	default:
		return "", fmt.Errorf("invalid input type")
	}
}

// processNumberArray converts a number array to a comma-separated string
func processNumberArray(numbers []interface{}) (string, error) {
	var result []string
	for _, num := range numbers {
		switch n := num.(type) {
		case float64:
			result = append(result, fmt.Sprintf("%g", n))
		case int:
			result = append(result, fmt.Sprintf("%d", n))
		default:
			return "", fmt.Errorf("invalid input array element type")
		}
	}
	return strings.Join(result, ","), nil
}
