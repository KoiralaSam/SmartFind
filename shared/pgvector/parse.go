package pgvector

import (
	"errors"
	"strconv"
	"strings"
)

// ParseLiteral parses a pgvector text literal like: [1,2,3] into a float32 slice.
func ParseLiteral(s string) ([]float32, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return nil, errors.New("empty vector literal")
	}
	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil, errors.New("invalid vector literal")
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]"))
	if inner == "" {
		return []float32{}, nil
	}

	parts := strings.Split(inner, ",")
	out := make([]float32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, errors.New("invalid vector literal element")
		}
		f, err := strconv.ParseFloat(p, 32)
		if err != nil {
			return nil, err
		}
		out = append(out, float32(f))
	}
	return out, nil
}
