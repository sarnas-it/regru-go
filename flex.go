package regru

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// Num — число, которое REG.API отдаёт то строкой ("200.00"), то числом (12345).
// Хранится как строка, чтобы не терять точность; для вычислений есть Float64 и Int.
type Num string

func (n *Num) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if string(b) == "null" {
		*n = ""
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*n = Num(s)
		return nil
	}
	*n = Num(b)
	return nil
}

func (n Num) String() string { return string(n) }

// Float64 разбирает значение; пустое и "null" дают 0.
func (n Num) Float64() (float64, error) {
	if n == "" || n == "null" {
		return 0, nil
	}
	return strconv.ParseFloat(string(n), 64)
}

// Int разбирает целое значение; пустое и "null" дают 0.
func (n Num) Int() (int, error) {
	if n == "" || n == "null" {
		return 0, nil
	}
	return strconv.Atoi(string(n))
}

// Flag — булево значение, которое приходит как "1"/"0", 1/0 или true/false.
type Flag bool

func (f *Flag) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	*f = s == "1" || s == "true"
	return nil
}

// List — список строк, который в одних методах приходит массивом, а в других
// строкой вида "a, b, c" или "[ 1, 2, 3 ]".
type List []string

func (l *List) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if string(b) == "null" {
		*l = nil
		return nil
	}
	if len(b) > 0 && b[0] == '[' {
		var raw []json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			return err
		}
		out := make(List, 0, len(raw))
		for _, r := range raw {
			var n Num
			if err := n.UnmarshalJSON(r); err != nil {
				return err
			}
			out = append(out, string(n))
		}
		*l = out
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	s = strings.Trim(strings.TrimSpace(s), "[]")
	*l = nil
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			*l = append(*l, p)
		}
	}
	return nil
}
