package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func valid() *Config {
	return &Config{
		NetBoxURL:   "https://netbox.example.com",
		NetBoxToken: "tok",
		PageSize:    50,
	}
}

func TestValidateOK(t *testing.T) {
	assert.NoError(t, valid().Validate())
}

func TestValidateMissingURL(t *testing.T) {
	c := valid()
	c.NetBoxURL = ""
	assert.ErrorContains(t, c.Validate(), "NETBOX_URL")
}

func TestValidateMissingToken(t *testing.T) {
	c := valid()
	c.NetBoxToken = ""
	assert.ErrorContains(t, c.Validate(), "NETBOX_TOKEN")
}

func TestValidateRelativeURL(t *testing.T) {
	c := valid()
	c.NetBoxURL = "netbox.example.com"
	assert.ErrorContains(t, c.Validate(), "absolute URL")
}

func TestValidatePageSize(t *testing.T) {
	c := valid()
	c.PageSize = 0
	assert.ErrorContains(t, c.Validate(), "page-size")
}

// With passthrough the token arrives per request, so a configured one is
// optional rather than required.
func TestValidateMissingTokenWithPassthrough(t *testing.T) {
	c := valid()
	c.NetBoxToken = ""
	c.TokenPassthrough = true
	assert.NoError(t, c.Validate())
}
