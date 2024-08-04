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

func DefaultIfNil[T any](t *T) T {
	if t == nil {
		var _default T
		return _default
	}
	return *t
}

func NilIfDefault[T comparable](t *T) *T {
	var _default T
	if t == nil {
		return nil
	}
	if *t == _default {
		return nil
	}
	return t
}
