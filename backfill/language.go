package backfill

// DeepCompare returns true if all values in a map[int][bool] are equal to the passed value.
func DeepCompare(items map[int]bool, value bool) bool {
	for _, flag := range items {
		if flag != value {
			return false
		}
	}
	return true
}
