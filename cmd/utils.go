package main

import "strings"

// Returns true if slice s contains element e, false otherwise.
func sliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Returns lowercased slice of strings.
func sliceLower(s []string) []string {
	r := []string{}
	for _, a := range s {
		r = append(r, strings.ToLower(a))
	}
	return r
}

// Returns slice s with all occurences of e removed.
func sliceRemove(s []string, e string) []string {
	var hit bool
	for {
		hit = false
		for i, a := range s {
			if a == e {
				s = append(s[:i], s[i+1:]...)
				hit = true
				break
			}
		}
		if !hit {
			break
		}
	}
	return s
}

// Returns slice s with duplicates removed.
func sliceDeduplicate(s []string) []string {
	keys := make(map[string]bool)
	r := []string{}
	for _, e := range s {
		if _, seen := keys[e]; !seen {
			keys[e] = true
			r = append(r, e)
		}
	}
	return r
}
