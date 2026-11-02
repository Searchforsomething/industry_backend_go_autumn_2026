package main

func rotateRunes(s string, shift int) string {
	if s == "" {
		return s
	}
	sRunes := []rune(s)
	shift = shift % len(sRunes)
	if shift < 0 {
		shift += len(sRunes)
	}
	result := string(sRunes[shift:]) + string(sRunes[:shift])
	return result
}
