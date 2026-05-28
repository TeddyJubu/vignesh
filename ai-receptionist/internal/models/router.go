package models

import (
	"os"
	"strings"
)

var defaultCfgModel string

// SetConfigModel sets the model from config.json (call once at startup from main).
func SetConfigModel(model string) {
	defaultCfgModel = strings.TrimSpace(model)
}

// GetModel resolves the model name for a task type.
func GetModel(taskType string) string {
	switch taskType {
	case "intent_classify":
		if m := strings.TrimSpace(os.Getenv("INTENT_CLASSIFY_MODEL")); m != "" {
			return m
		}
		if defaultCfgModel != "" {
			return defaultCfgModel
		}
		return "gemma4:31b-cloud"
	default:
		if defaultCfgModel != "" {
			return defaultCfgModel
		}
		return "gemma4:31b-cloud"
	}
}
