// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/api"
)

const (
	HeaderXScopeOrgID = "X-Scope-OrgID"
)

func newMimirRoundTripper(mimirScopeOrgID *string) *mimirRoundTripper {
	return &mimirRoundTripper{rt: api.DefaultRoundTripper, mimirScopeOrgID: mimirScopeOrgID}
}

type mimirRoundTripper struct {
	rt              http.RoundTripper
	mimirScopeOrgID *string
}

// RoundTrip adds additional HTTP headers required by Mimir.
func (s mimirRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	//nolint:staticcheck // ignore SA1008 rule to avoid headers duplication
	if len(r.Header.Get(HeaderXScopeOrgID)) == 0 && len(r.Header[HeaderXScopeOrgID]) == 0 {
		// The specification of http.RoundTripper says that it shouldn't mutate
		// the request so make a copy of req.Header since this is all that is
		// modified.
		r2 := new(http.Request)
		*r2 = *r
		r2.Header = make(http.Header)
		for k, v := range r.Header {
			r2.Header[k] = v
		}
		r2.Header.Set(HeaderXScopeOrgID, *s.mimirScopeOrgID)
		r = r2
	}
	return s.rt.RoundTrip(r)
}
