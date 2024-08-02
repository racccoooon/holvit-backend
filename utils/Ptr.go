package utils

func Ptr[T any](t T) *T {
	return &t
}

func GetOrDefault[T any](t *T, _default T) T {
	if t == nil {
		return _default
	}
	return *t
}
