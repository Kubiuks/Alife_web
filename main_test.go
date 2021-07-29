package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponse(t *testing.T) {

	req, err := http.NewRequest("GET", "", nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()

	hf := http.HandlerFunc(agentsHandler)
	hf.ServeHTTP(w, req)

	fmt.Printf("%d - %s", w.Code, w.Body.String())
}
