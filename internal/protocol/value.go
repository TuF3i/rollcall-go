package protocol

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// ParseInt converts loosely-typed decoded JSON values into int.
func ParseInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		if math.Trunc(float64(n)) != float64(n) {
			return 0, false
		}
		return int(n), true
	case float64:
		if math.Trunc(n) != n {
			return 0, false
		}
		return int(n), true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i), true
		}
		f, err := n.Float64()
		if err != nil || math.Trunc(f) != f {
			return 0, false
		}
		return int(f), true
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		if i, err := strconv.Atoi(s); err == nil {
			return i, true
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || math.Trunc(f) != f {
			return 0, false
		}
		return int(f), true
	default:
		return 0, false
	}
}
