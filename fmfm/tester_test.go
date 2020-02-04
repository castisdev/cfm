package fmfm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestFsNotify(t *testing.T) {
	assert.Equal(t, true, TestFsNotify())
}
