package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserIDAttributeIsMasked(t *testing.T) {
	attr := userIDAttribute("warden.user.id", "0123456789abcdef")
	assert.Equal(t, "warden.user.id", string(attr.Key))
	assert.Equal(t, "0123************", attr.Value.AsString())
}
