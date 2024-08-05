package h

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Some(t *testing.T) {
	// arrange
	value := 3

	// act
	out := Some(value)

	// assert
	assert.True(t, out.IsSome())
	assert.False(t, out.IsNone())

	assert.Equal(t, value, out.Expect(""))
	assert.Equal(t, value, out.Unwrap())
	assert.Equal(t, value, out.UnwrapErr(errors.New("")))
	assert.Equal(t, value, out.UnwrapOr(5))
	assert.Equal(t, value, out.UnwrapOrElse(func() int {
		return 1
	}))
	assert.Equal(t, value, out.UnwrapOrEmpty())
}

func Test_None(t *testing.T) {
	// act
	out := None[int]()

	// assert
	assert.False(t, out.IsSome())
	assert.True(t, out.IsNone())

	assert.Panics(t, func() {
		out.Expect("correct")
	})
	assert.Panics(t, func() {
		out.Unwrap()
	})
	assert.Panics(t, func() {
		out.UnwrapErr(errors.New(""))
	})
	assert.Equal(t, 5, out.UnwrapOr(5))
	assert.Equal(t, 1, out.UnwrapOrElse(func() int {
		return 1
	}))
	assert.Equal(t, 0, out.UnwrapOrEmpty())
}
