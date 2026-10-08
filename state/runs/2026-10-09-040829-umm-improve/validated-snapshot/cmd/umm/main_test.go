package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/hkjang/umm/internal/dream"
)

func TestHTTPWriteTimeoutCoversMaximumGatewayBudget(t *testing.T) {
	server := newHTTPServer(":0", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	minimum := time.Duration(dream.MaxGatewayTimeoutSeconds)*time.Second + httpWriteTimeoutBuffer
	if server.WriteTimeout < minimum {
		t.Fatalf("write timeout %s is shorter than maximum gateway budget %s", server.WriteTimeout, minimum)
	}
}
