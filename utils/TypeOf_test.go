package utils

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func Test_int(t *testing.T) {
	// arrange
	var v int
	expected := reflect.TypeOf(v)

	// act
	result := TypeOf[int]()

	// assert
	assert.Equal(t, expected, result)
}

func Test_string(t *testing.T) {
	// arrange
	var v string
	expected := reflect.TypeOf(v)

	// act
	result := TypeOf[string]()

	// assert
	assert.Equal(t, expected, result)
}

type TestStruct struct{}

func Test_struct(t *testing.T) {
	// arrange
	var v TestStruct
	expected := reflect.TypeOf(v)

	// act
	result := TypeOf[TestStruct]()

	// assert
	assert.Equal(t, expected, result)
}
