package main

import (
	"encoding/json"
	"net/http"
)

type httpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func writeJSONError(w http.ResponseWriter, code int, message string, details ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	err := httpError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}

	json.NewEncoder(w).Encode(err)
}