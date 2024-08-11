package h

import (
	"errors"
	"fmt"
)

type Opt[T any] struct {
	value *T
}

func (o Opt[T]) String() string {
	if o.value == nil {
		return "<none>"
	} else {
		return fmt.Sprintf("%v", *o.value)
	}
}

func Some[T any](v T) Opt[T] {
	return Opt[T]{
		value: &v,
	}
}

func None[T any]() Opt[T] {
	return Opt[T]{}
}

func SomeIf[T any](cond bool, t T) Opt[T] {
	if cond {
		return Some(t)
	}
	return None[T]()
}

func FromPtr[T any](p *T) Opt[T] {
	return Opt[T]{
		value: p,
	}
}

func FromDefault[T comparable](v T) Opt[T] {
	var zero T
	if v == zero {
		return Opt[T]{}
	}
	return Opt[T]{value: &v}
}

func (o Opt[T]) IsNone() bool {
	return o.value == nil
}

func (o Opt[T]) IsSome() bool {
	return o.value != nil
}

func (o Opt[T]) Get() (T, bool) {
	if o.IsNone() {
		var zero T
		return zero, false
	} else {
		return *o.value, true
	}
}

func (o Opt[T]) And(other Opt[T]) Opt[T] {
	if o.value == nil {
		return o
	}
	return other
}

func (o Opt[T]) AndThen(fn func(T) Opt[T]) Opt[T] {
	if o.value == nil {
		return o
	}
	return fn(*o.value)
}

func (o Opt[T]) Or(other Opt[T]) Opt[T] {
	if o.value == nil {
		return other
	}
	return o
}

func (o Opt[T]) OrDefault(t T) T {
	if o.value == nil {
		return t
	}
	return *o.value
}

func (o Opt[T]) OrElse(fn func() Opt[T]) Opt[T] {
	if o.value != nil {
		return o
	}
	return fn()
}

func (o Opt[T]) OrElseDefault(fn func() T) T {
	if o.value != nil {
		return *o.value
	}
	return fn()
}

func (o Opt[T]) ToNillablePtr() *T {
	return o.value
}

func (o *Opt[T]) AsMutPtr() **T {
	return &o.value
}

func (o Opt[T]) Expect(msg string) T {
	return o.UnwrapErr(fmt.Errorf("tried to unwrap an empty option: %s", msg))
}

func (o Opt[T]) Unwrap() T {
	return o.UnwrapErr(errors.New("tried to unwrap an empty option"))
}

func (o Opt[T]) UnwrapErr(e error) T {
	if o.value == nil {
		panic(e)
	}
	return *o.value
}

func (o Opt[T]) UnwrapOr(d T) T {
	if o.value == nil {
		return d
	}
	return *o.value
}

func (o Opt[T]) UnwrapOrElse(f func() T) T {
	if o.value == nil {
		return f()
	}
	return *o.value
}

func (o Opt[T]) UnwrapOrEmpty() T {
	if o.value == nil {
		var zero T
		return zero
	}
	return *o.value
}

func (o Opt[T]) Map(m func(T) T) Opt[T] {
	if o.value == nil {
		return o
	}
	v := m(*o.value)
	return Opt[T]{
		value: &v,
	}
}

func MapOpt[From any, To any](from Opt[From], m func(From) To) Opt[To] {
	if from.value == nil {
		return Opt[To]{}
	}
	mapped := m(*from.value)
	return Opt[To]{value: &mapped}
}

func (o Opt[T]) IfSome(f func(T)) {
	if o.value == nil {
		return
	}
	f(*o.value)
}
