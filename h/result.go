package h

type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](value T) Result[T] {
	return Result[T]{
		value: value,
		err:   nil,
	}
}

func Err[T any](err error) Result[T] {
	var zero T
	return Result[T]{
		value: zero,
		err:   err,
	}
}

func (r Result[T]) IsOk() bool {
	return r.err == nil
}

func (r Result[T]) IsErr() bool {
	return r.err != nil
}

func (r Result[T]) Unwrap() T {
	if r.IsErr() {
		panic(r.err)
	}
	return r.value
}

func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.IsErr() {
		return defaultValue
	}
	return r.value
}

func (r Result[T]) UnwrapErr() error {
	if r.IsOk() {
		panic("called UnwrapErr on a successful Result")
	}
	return r.err
}

func (r Result[T]) Match(ok func(T), err func(error)) {
	if r.IsOk() {
		ok(r.value)
	} else {
		err(r.err)
	}
}
