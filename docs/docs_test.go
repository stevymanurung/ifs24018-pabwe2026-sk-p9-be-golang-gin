package docs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSwaggerJSON(t *testing.T) {
	doc, err := GetSwaggerJSON("localhost:8081", "http")
	assert.NoError(t, err)
	assert.NotEmpty(t, doc)
	assert.Contains(t, doc, "localhost:8081")

	doc2, err := GetSwaggerJSON("", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, doc2)
}
