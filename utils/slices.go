package utils

func IsSliceSubset[T comparable](set, subset []T) bool {
	setMap := make(map[T]struct{})
	for _, v := range set {
		setMap[v] = struct{}{}
	}

	for _, v := range subset {
		if _, found := setMap[v]; !found {
			return false
		}
	}

	return true
}

func RemoveFirst[T any](slice []T, fn func(T) bool) []T {
	// Find the index of the value to remove
	index := -1
	for i, v := range slice {
		if fn(v) {
			index = i
			break
		}
	}

	if index == -1 {
		// Value not found, return the original slice
		return slice
	}

	// Remove the entry at the found index
	return append(slice[:index], slice[index+1:]...)
}
