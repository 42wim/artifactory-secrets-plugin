package artifactory

import (
	"bytes"
	"context"
	"io/ioutil"
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

// Literally https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-CreateToken
const canonicalAccessToken = `{
   "access_token":   "eyXsdgbtybbeeyh...",
   "expires_in":    0,
   "scope":         "api:* member-of-groups:example",
   "token_type":    "Bearer",
   "refresh_token": "fgsfgsdugh8dgu9s8gy9hsg..."
}`

const artVersion = `{
    "version": "7.19.10",
    "revision": "71910900",
    "license": "05179b957028fa9aa1ceb88da6519a245e55b9fc5"
}`

func tokenCreatedResponse(token string) roundTripperFunc {
	return func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(token)),
		}, nil
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
