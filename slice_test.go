package dev_test

import (
	"testing"

	"github.com/wish/dev"
)

func TestContainsString(t *testing.T) {
	tests := []struct {
		Input       []string
		InputString string
		Expected    bool
	}{
		{[]string{"foo", "bar", "baz"}, "foo", true},
		{[]string{"foo", "bar", "baz"}, "bazl", false},
		{[]string{"foo", "bar", "baz"}, "", false},
		{[]string{"foo", "", "baz"}, "", true},
		{[]string{""}, "foo", false},
	}

	for _, test := range tests {
		contains := dev.SliceContainsString(test.Input, test.InputString)
		if contains != test.Expected {
			t.Errorf("SliceContainsString returned %t, but expected %t", contains, test.Expected)
		}
	}
}

func TestSliceInsertString(t *testing.T) {
	tests := []struct {
		ExistingSlice []string
		InputString   string
		Position      int
		Expected      []string
	}{
		{[]string{"foo"}, "far", 0, []string{"far", "foo"}},
		{[]string{"foo"}, "far", 1, []string{"foo", "far"}},
		{[]string{""}, "far", 0, []string{"far", ""}},
		{[]string{""}, "", 0, []string{"", ""}},
	}

	for _, test := range tests {
		newSlice := dev.SliceInsertString(test.ExistingSlice, test.InputString, test.Position)
		if len(newSlice) != len(test.Expected) {
			t.Errorf("Incorrect length of returned slice, got %v: %d, expected %v:%d", newSlice, len(newSlice), test.Expected, len(test.Expected))
		}
		for i, str := range test.Expected {
			if newSlice[i] != str {
				t.Errorf("Expected value of string to be '%s' but got '%s'", str, newSlice[i])
			}
		}
	}
}
