package utils

import (
	"encoding/json"
	"fmt"
)

func ToString(v any) (string, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%f", v), nil
	}
	bs, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("unsupport format: %w", err)
	}
	return string(bs), nil
}
