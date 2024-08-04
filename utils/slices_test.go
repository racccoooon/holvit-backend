package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_IsSliceSubset_BothEmpty(t *testing.T) {
	// arrange
	s1 := []int{}
	s2 := []int{}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.True(t, result)
}

func Test_IsSliceSubset_SetEmpty(t *testing.T) {
	// arrange
	s1 := []int{}
	s2 := []int{1}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.False(t, result)
}

func Test_IsSliceSubset_SusbetEmpty(t *testing.T) {
	// arrange
	s1 := []int{1}
	s2 := []int{}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.True(t, result)
}

func Test_IsSliceSubset_NonEmptyEqualSingleItem(t *testing.T) {
	// arrange
	s1 := []int{1}
	s2 := []int{1}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.True(t, result)
}

func Test_IsSliceSubset_NonEmptyEqualMultipleItems(t *testing.T) {
	// arrange
	s1 := []int{1, 2, 3}
	s2 := []int{1, 3, 2}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.True(t, result)
}

func Test_IsSliceSubset_SmallerSubsetMatches(t *testing.T) {
	// arrange
	s1 := []int{1, 2, 3}
	s2 := []int{1, 3}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.True(t, result)
}

func Test_IsSliceSubset_SmallerSubsetDoesntMatch(t *testing.T) {
	// arrange
	s1 := []int{1, 2, 3}
	s2 := []int{1, 4}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.False(t, result)
}

func Test_IsSliceSubset_EmptySubset(t *testing.T) {
	// arrange
	s1 := []int{1, 2, 3}
	s2 := []int{}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.True(t, result)
}

func Test_IsSliceSubset_EmptySet(t *testing.T) {
	// arrange
	s1 := []int{}
	s2 := []int{1, 2, 3}

	// act
	result := IsSliceSubset(s1, s2)

	// arrange
	assert.False(t, result)
}
