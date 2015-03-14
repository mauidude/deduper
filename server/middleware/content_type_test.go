package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentType(t *testing.T) {
	ct := ContentType{Type: "text/plain"}
	rw := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/asdf", nil)
	next := &mockHandler{}

	ct.ServeHTTP(rw, r, next.ServeHTTP)

	assert.Equal(t, "text/plain", rw.Header().Get("Content-Type"))
	assert.True(t, next.called)
}
