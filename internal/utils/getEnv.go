package utils

import (
	"fmt"
	"os"
)

func GetWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetWithDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		_, err := fmt.Sscanf(value, "%d", &intValue)
		if err == nil {
			return intValue
		}
	}
	return defaultValue
}
func GetWithDefaultFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var intValue float64
		_, err := fmt.Sscanf(value, "%f", &intValue)
		if err == nil {
			return intValue
		}
	}
	return defaultValue
}
