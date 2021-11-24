package gateway

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HTTPRequestConversion(t *testing.T) {

	input := "/blah%2fhello"

	req, err := http.NewRequest(http.MethodGet, "http://blah"+input, nil)
	require.NoError(t, err)

	converted := convertHTTPRequest(req)
	assert.Equal(t, input, converted.Path)

}
