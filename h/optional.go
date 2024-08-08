package h

import (
	"errors"
	"fmt"
)

type Optional[T any] struct {
	value *T
}

func (o Optional[T]) String() string {
	if o.value == nil {
		return "<none>"
	} else {
		return fmt.Sprintf("%v", *o.value)
	}
}

func Some[T any](v T) Optional[T] {
	return Optional[T]{
		value: &v,
	}
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func FromPtr[T any](p *T) Optional[T] {
	return Optional[T]{
		value: p,
	}
}

func FromDefault[T comparable](v T) Optional[T] {
	var zero T
	if v == zero {
		return Optional[T]{}
	}
	return Optional[T]{value: &v}
}

func (o Optional[T]) IsNone() bool {
	return o.value == nil
}

func (o Optional[T]) IsSome() bool {
	return o.value != nil
}

func (o Optional[T]) Get() (T, bool) {
	if o.IsNone() {
		var zero T
		return zero, false
	} else {
		return *o.value, true
	}
}

func (o Optional[T]) And(other Optional[T]) Optional[T] {
	if o.value == nil {
		return o
	}
	return other
}

func (o Optional[T]) AndThen(fn func(T) Optional[T]) Optional[T] {
	if o.value == nil {
		return o
	}
	return fn(*o.value)
}

func (o Optional[T]) Or(other Optional[T]) Optional[T] {
	if o.value == nil {
		return other
	}
	return o
}

func (o Optional[T]) OrDefault(t T) T {
	if o.value == nil {
		return t
	}
	return *o.value
}

func (o Optional[T]) OrElse(fn func() Optional[T]) Optional[T] {
	if o.value != nil {
		return o
	}
	return fn()
}

func (o Optional[T]) OrElseDefault(fn func() T) T {
	if o.value != nil {
		return *o.value
	}
	return fn()
}

func (o Optional[T]) ToNillablePtr() *T {
	return o.value
}

func (o *Optional[T]) AsMutPtr() **T {
	return &o.value
}

func (o Optional[T]) Expect(msg string) T {
	return o.UnwrapErr(errors.New(fmt.Sprintf("tried to unwrap an empty option: %s", msg)))
}

func (o Optional[T]) Unwrap() T {
	return o.UnwrapErr(errors.New("tried to unwrap an empty option"))
}

func (o Optional[T]) UnwrapErr(e error) T {
	if o.value == nil {
		panic(e)
	}
	return *o.value
}

func (o Optional[T]) UnwrapOr(d T) T {
	if o.value == nil {
		return d
	}
	return *o.value
}

func (o Optional[T]) UnwrapOrElse(f func() T) T {
	if o.value == nil {
		return f()
	}
	return *o.value
}

func (o Optional[T]) UnwrapOrEmpty() T {
	if o.value == nil {
		var zero T
		return zero
	}
	return *o.value
}

func (o Optional[T]) Map(m func(T) T) Optional[T] {
	if o.value == nil {
		return o
	}
	v := m(*o.value)
	return Optional[T]{
		value: &v,
	}
}

func MapOpt[From any, To any](from Optional[From], m func(From) To) Optional[To] {
	if from.value == nil {
		return Optional[To]{}
	}
	mapped := m(*from.value)
	return Optional[To]{value: &mapped}
}

func (o Optional[T]) IfSome(f func(T)) {
	if o.value == nil {
		return
	}
	f(*o.value)
}
