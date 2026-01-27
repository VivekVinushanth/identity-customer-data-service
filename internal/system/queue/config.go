package queue

import "fmt"

func GetString(cfg map[string]any, key, def string) (string, error) {
	v, ok := cfg[key]
	if !ok {
		return def, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("config key %s must be string", key)
	}
	return s, nil
}

func GetInt(cfg map[string]any, key string, def int) (int, error) {
	v, ok := cfg[key]
	if !ok {
		return def, nil
	}
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case float64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("config key %s must be number", key)
	}
}
