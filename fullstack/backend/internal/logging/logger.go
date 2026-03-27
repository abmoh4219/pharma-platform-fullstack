package logging

import (
	"encoding/json"
	"log"
	"sort"
	"strings"
)

func Log(level, category, message string, fields map[string]any) {
	if strings.TrimSpace(level) == "" {
		level = "INFO"
	}
	if strings.TrimSpace(category) == "" {
		category = "general"
	}

	payload := map[string]any{
		"level":    strings.ToUpper(strings.TrimSpace(level)),
		"category": strings.TrimSpace(category),
		"message":  strings.TrimSpace(message),
	}
	for k, v := range fields {
		payload[k] = v
	}

	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make(map[string]any, len(payload))
	for _, k := range keys {
		ordered[k] = payload[k]
	}
	buf, _ := json.Marshal(ordered)
	log.Print(string(buf))
}

func Info(category, message string, fields map[string]any) {
	Log("INFO", category, message, fields)
}

func Warn(category, message string, fields map[string]any) {
	Log("WARN", category, message, fields)
}

func Error(category, message string, fields map[string]any) {
	Log("ERROR", category, message, fields)
}
