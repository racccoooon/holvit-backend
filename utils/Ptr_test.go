package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Ptr(t *testing.T) {
	// arrange
	v := 3

	// act
	p := Ptr(v)

	// assert
	assert.Equal(t, p, &v)
}

func Test_GetOrDefault_Nil(t *testing.T) {
	// arrange
	var p *int = nil

	// act
	result := GetOrDefault(p, 69)

	// assert
	assert.Equal(t, result, 69)
}

func Test_GetOrDefault_NotNil(t *testing.T) {
	// arrange
	expected := 3
	p := &expected

	// act
	result := GetOrDefault(p, 69)

	// assert
	assert.Equal(t, result, expected)
}
