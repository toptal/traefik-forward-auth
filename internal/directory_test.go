package tfa

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

/**
 * Tests
 */

func TestDirectory(t *testing.T) {
	assert := assert.New(t)
	config = newDefaultConfig()
	config.GoogleGroups = []string{"core@toptal.com"}
	config.GoogleDomain = "toptal.com"
	config.GoogleApplicationCredentials = "/tmp/credentials.json"
	config.GoogleActingAdminEmail = "services-core@toptal.com"
	config.GoogleExpirySeconds = 10
	directory := NewDirectory()
	assert.Empty(directory.cache, "cache should be empty")
	assert.Empty(directory.ttl, "ttl should be empty")
	assert.True(directory.IsMember("jonathan.doveston@toptal.com", "core@toptal.com"), "should be a member")
	assert.False(directory.IsMember("jonathan.doveston@toptal.com", "billing-team@toptal.com"), "should not be a member")
	assert.NotEmpty(directory.cache, "cache should not be empty")
	assert.NotEmpty(directory.ttl, "ttl should not be empty")
}
