package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	r, err := Add(2, 3)
	assert.NoError(t, err)
	assert.Equal(t, 5, r)
}
