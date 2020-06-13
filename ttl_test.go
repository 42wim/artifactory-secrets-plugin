package artifactory

import (
	"bytes"
	"context"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

// I've centralized all the tests involving the interplay of TTLs.

// Backend max ttl cannot exceed system max ttl. If it is, the config will fail.
func TestBackend_MaxTTLNotGreaterThanSystem(t *testing.T) {
	b, config := makeBackend(t)

	exceedsSystemTTL := map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
		"max_ttl":      logical.TestBackendConfig().System.MaxLeaseTTL() + time.Second,
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config/admin",
		Storage:   config.StorageView,
		Data:      exceedsSystemTTL,
	})

	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.True(t, resp.IsError())
}

// If default_ttl for the backend is higher than the system max_ttl, then it is lowered to the system max_ttl.
func TestBackend_BackendDefaultTTLNotGreaterThanSystemDefaultTTL(t *testing.T) {
	b, config := makeBackend(t)

	exceedsSystemTTL := map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
		"default_ttl":  config.System.MaxLeaseTTL() + time.Second,
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config/admin",
		Storage:   config.StorageView,
		Data:      exceedsSystemTTL,
	})

	assert.NotNil(t, resp)
	assert.False(t, resp.IsError())
	assert.NotEmpty(t, resp.Warnings)
	assert.NoError(t, err)

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config/admin",
		Storage:   config.StorageView,
	})
	assert.NoError(t, err)
	assert.EqualValues(t, config.System.MaxLeaseTTL().Seconds(), resp.Data["default_ttl"])
}

// If the default_ttl for the backend is set higher than the max_ttl, then it is lowered to the max_ttl.
func TestBackend_DefaultTTLNotGreaterThanBackendMaxTTL(t *testing.T) {
	b, config := makeBackend(t)

	exceedsSystemTTL := map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
		"max_ttl":      5 * time.Minute,
		"default_ttl":  (5 * time.Minute) + time.Second,
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config/admin",
		Storage:   config.StorageView,
		Data:      exceedsSystemTTL,
	})

	assert.NotNil(t, resp)
	assert.False(t, resp.IsError())
	assert.NotEmpty(t, resp.Warnings)
	assert.NoError(t, err)

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config/admin",
		Storage:   config.StorageView,
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 5*time.Minute.Seconds(), resp.Data["default_ttl"])
}

// Both default_ttl and max_ttl can equal the system-wide max_ttl though.
func TestBackend_TTLsCanEqualSystemTTL(t *testing.T) {

	b, config := makeBackend(t)

	exceedsSystemTTL := map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
		"default_ttl":  config.System.MaxLeaseTTL(),
		"max_ttl":      config.System.MaxLeaseTTL(),
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config/admin",
		Storage:   config.StorageView,
		Data:      exceedsSystemTTL,
	})
	assert.Nil(t, resp)
	assert.NoError(t, err)
}

// Role max_ttl cannot exceed backend max_ttl. The role will fail to be created.
func TestBackend_RoleTTLNotGreaterBackendMaxTTL(t *testing.T) {
	b, config := configuredBackend(t, map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
		"max_ttl":      5 * time.Minute,
	})

	roleData := map[string]interface{}{
		"role":     "test-role",
		"username": "test-username",
		"scope":    "test-scope",
		"max_ttl":  5*time.Minute + time.Second,
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/test-role",
		Storage:   config.StorageView,
		Data:      roleData,
	})

	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.True(t, resp.IsError())
}

// If no max ttl is set, then the system max ttl is used, so default_ttl can equal system max_ttl without issue.
func TestBackend_UseSystemMaxTTL(t *testing.T) {
	b, config := configuredBackend(t, map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
		"default_ttl":  logical.TestBackendConfig().System.MaxLeaseTTL(),
	})
	assert.NotNil(t, b)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config/admin",
		Storage:   config.StorageView,
	})
	assert.NoError(t, err)
	assert.EqualValues(t, config.System.MaxLeaseTTL().Seconds(), resp.Data["default_ttl"])
}

// Role default_ttl is higher than max_ttl, then role creation will fail.
func TestBackend_RoleDefaultTTLNotGreaterThanRoleMaxTTL(t *testing.T) {
	b, config := configuredBackend(t, map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
		"max_ttl":      5 * time.Minute,
	})

	roleData := map[string]interface{}{
		"role":        "test-role",
		"username":    "test-username",
		"scope":       "test-scope",
		"max_ttl":     5 * time.Minute,
		"default_ttl": 5*time.Minute + time.Second,
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/test-role",
		Storage:   config.StorageView,
		Data:      roleData,
	})
	assert.NotNil(t, resp)
	assert.True(t, resp.IsError())
	assert.NoError(t, err)
}

// Role with no Max TTL must use the system max TTL when creating tokens.
func TestBackend_NoRoleMaxTTLUsesSystemMaxTTL(t *testing.T) {
	b, config := configuredBackend(t, map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
	})

	// Role with no maximum TTL
	roleData := map[string]interface{}{
		"role":     "test-role",
		"username": "test-username",
		"scope":    "test-scope",
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/test-role",
		Storage:   config.StorageView,
		Data:      roleData,
	})
	assert.Nil(t, resp)
	assert.NoError(t, err)

	// Fake the Artifactory response
	b.httpClient = newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(bytes.NewBufferString(`
{
   "access_token":   "adsdgbtybbeeyh...",
   "expires_in":    3600,
   "scope":         "api:* member-of-groups:readers",
   "token_type":    "Bearer",
   "refresh_token": "fgsfgsdugh8dgu9s8gy9hsg..."
}
`)),
			Header: make(http.Header),
		}, nil
	})

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "token/test-role",
		Storage:   config.StorageView,
	})
	assert.NotNil(t, resp)
	assert.NoError(t, err)

	assert.EqualValues(t, config.System.MaxLeaseTTL(), resp.Secret.MaxTTL)
}

// With a role max_ttl not greater than the system max_ttl, the max_ttl for a token must
// be the role max_ttl.
func TestBackend_WorkingWithBothMaxTTLs(t *testing.T) {
	b, config := configuredBackend(t, map[string]interface{}{
		"access_token": "test-access-token",
		"url":          "https://127.0.0.1",
		"max_ttl":      10 * time.Minute,
	})

	// Role with no maximum TTL
	roleData := map[string]interface{}{
		"role":        "test-role",
		"username":    "test-username",
		"scope":       "test-scope",
		"refreshable": true,
		"max_ttl":     9 * time.Minute,
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/test-role",
		Storage:   config.StorageView,
		Data:      roleData,
	})
	assert.Nil(t, resp)
	assert.NoError(t, err)

	b.httpClient = newTestClient(tokenCreatedResponse(canonicalAccessToken))

	resp, err = b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "token/test-role",
		Storage:   config.StorageView,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.IsError())

	assert.EqualValues(t, 9*time.Minute, resp.Secret.MaxTTL)
}
