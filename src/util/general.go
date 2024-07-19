package util

// Times runs <iteratee> <count> amount of times.
//
// If <iteratee> returns false, iteration stops.
//
// <iteration> argument starts from 1 until it equals <count>.
func Times(count int, iteratee func(iteration int) bool) {
	for iteration := 1; iteration <= count; iteration++ {
		if !iteratee(iteration) {
			break
		}
	}
}
