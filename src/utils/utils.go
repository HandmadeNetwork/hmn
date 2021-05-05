package utils

func IntMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func IntMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func IntClamp(min, t, max int) int {
	return IntMax(min, IntMin(t, max))
}
