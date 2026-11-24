package middleware

import "net/http"

func WriteProxyError(w http.ResponseWriter, status int, message string) {
	writeJSONError(w, status, message)
}
