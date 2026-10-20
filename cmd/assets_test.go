package cmd

import (
	"net/http"
	"testing"
)

func TestBuildsClientHasNoTotalTimeout(t *testing.T) {
	c := (&options{}).builds().HTTP
	tr, ok := c.Transport.(*http.Transport)
	if c.Timeout != 0 || !ok || tr.ResponseHeaderTimeout == 0 || tr.TLSHandshakeTimeout == 0 || tr.Proxy == nil {
		t.Errorf("client = %+v", c)
	}
}
