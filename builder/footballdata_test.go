package main

import (
	"encoding/json"
	"testing"
)

func TestFdMinuteUnmarshal(t *testing.T) {
	cases := []struct {
		name string
		json string
		want *int
	}{
		{"integer", `{"minute": 56}`, intPtr(56)},
		{"null", `{"minute": null}`, nil},
		{"absent", `{}`, nil},
		{"string number", `{"minute": "56"}`, intPtr(56)},
		{"unparseable string", `{"minute": "45+2"}`, nil},
		{"wrong type", `{"minute": {"x": 1}}`, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got struct {
				Minute fdMinute `json:"minute"`
			}
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			switch {
			case tc.want == nil && got.Minute.v != nil:
				t.Errorf("minute = %d, want nil", *got.Minute.v)
			case tc.want != nil && (got.Minute.v == nil || *got.Minute.v != *tc.want):
				t.Errorf("minute = %v, want %d", got.Minute.v, *tc.want)
			}
		})
	}
}

func intPtr(n int) *int { return &n }
