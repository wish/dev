package dev

// SliceContainsString checks the provided slice for the specified
// string and returns true if it was found, otherwise returns false.
func SliceContainsString(slice []string, a string) bool {
	for _, b := range slice {
		if b == a {
			return true
		}
	}
	return false
}

// SliceInsertString inserts string at the specified index of the provided
// slice. From Slice Tricks.
func SliceInsertString(s []string, str string, index int) []string {
	s = append(s, "")
	copy(s[index+1:], s[index:])
	s[index] = str
	return s
}
