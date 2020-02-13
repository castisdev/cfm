package fmfm

import (
	"testing"

	"github.com/castisdev/cfm/myinotify"
	"github.com/stretchr/testify/assert"
)

func TestTestNotify(t *testing.T) {
	_, err := myinotify.NewWatcher()
	if err != nil {
		t.Skip("skip i-notify test")
	}
	assert.Equal(t, true, TestNotify())
}
