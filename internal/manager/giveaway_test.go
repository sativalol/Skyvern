package manager

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		err      bool
	}{
		{"2h", 2 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"30", 30 * time.Minute, false},
		{"invalid", 0, true},
	}

	for _, tc := range tests {
		got, err := ParseDuration(tc.input)
		if (err != nil) != tc.err {
			t.Errorf("ParseDuration(%q) unexpected error state: %v", tc.input, err)
			continue
		}
		if !tc.err && got != tc.expected {
			t.Errorf("ParseDuration(%q) = %v; want %v", tc.input, got, tc.expected)
		}
	}
}

func TestSnowflakeToTime(t *testing.T) {
	tm, err := SnowflakeToTime("175928847299117063")
	if err != nil {
		t.Fatalf("SnowflakeToTime failed: %v", err)
	}
	expected := time.Date(2016, time.April, 30, 11, 18, 25, 796000000, time.UTC)
	if tm.UTC() != expected {
		t.Errorf("SnowflakeToTime got %v, expected %v", tm.UTC(), expected)
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"red", 0xFF0000},
		{"#00ff00", 0x00FF00},
		{"blue", 0x0000FF},
		{"#123456", 0x123456},
		{"invalid", 0x2b2d31},
	}

	for _, tc := range tests {
		got := ParseColor(tc.input)
		if got != tc.expected {
			t.Errorf("ParseColor(%q) = 0x%X; want 0x%X", tc.input, got, tc.expected)
		}
	}
}
