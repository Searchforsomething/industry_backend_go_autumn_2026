package main

import "strings"

func greet(name string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return "Hello, World!"
	}

	return "Hello, " + trimmedName + "!"
}
