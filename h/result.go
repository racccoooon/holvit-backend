package h

import "fmt"

type Unit struct{}

type Result[T any] struct {
	value T
	err   error
}

type UResult = Result[Unit]

func UOk() Result[Unit] {
	return Ok(Unit{})
}

func Ok[T any](value T) Result[T] {
	return Result[T]{
		value: value,
		err:   nil,
	}
}

func UErr(err error) Result[Unit] {
	return Err[Unit](err)
}

func UErrIf(cond bool, err error) Result[Unit] {
	if cond {
		return UErr(err)
	}
	return UOk()
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
		panic(fmt.Errorf("called UnwrapErr on a successful Result"))
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

func (r Result[T]) MapErr(err func(error) error) Result[T] {
	if r.IsErr() {
		return Err[T](err(r.err))
	}
	return r
}

func (r Result[T]) SetErr(err error) Result[T] {
	if r.IsErr() {
		return Err[T](err)
	}
	return r
}

func MapResult[T1 any, T2 any](result Result[T1], mapping func(T1) T2) Result[T2] {
	if result.IsOk() {
		return Ok(mapping(result.Unwrap()))
	}
	return Err[T2](result.err)
}
