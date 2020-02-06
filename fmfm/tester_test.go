package fmfm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestNotify(t *testing.T) {
	assert.Equal(t, true, TestNotify())
}
