package artifactory

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (rt roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}

func newTestClient(fn roundTripperFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func makeBackend(t *testing.T) (*backend, *logical.BackendConfig) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}

	b, err := Backend(config)
	if err != nil {
		t.Fatal(err)
	}

	if err := b.Setup(context.Background(), config); err != nil {
		t.Fatal(err)
	}

	return b, config
}

func configuredBackend(t *testing.T, adminConfig map[string]interface{}) (*backend, *logical.BackendConfig) {

	b, config := makeBackend(t)

	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config/admin",
		Storage:   config.StorageView,
		Data:      adminConfig,
	})
	assert.NoError(t, err)

	return b, config
}
