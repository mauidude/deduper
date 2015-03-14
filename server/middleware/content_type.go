package middleware

import "net/http"

// ContentType middleware provides a content type to the HTTP ResponseWriter.
type ContentType struct {
	// Type will be the value of the Content-Type header on the ResponseWriter.
	Type string
}

// ServeHTTP applies the Content-Type header to the ResponseWriter.
func (c *ContentType) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	w.Header().Add("Content-Type", c.Type)
	next(w, r)
}
