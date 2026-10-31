package service

const MaxCreditsPerSemester = 24

func WithinCreditLimit(taken, adding int) bool {
	return taken+adding <= MaxCreditsPerSemester
}

func CreditsAfterSwap(taken, leaving, joining int) int {
	return taken - leaving + joining
}
