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
