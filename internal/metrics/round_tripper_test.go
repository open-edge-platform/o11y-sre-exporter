// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockRoundTrip struct {
	t              *testing.T
	expectedHeader http.Header
}

func (m *mockRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	require.Equal(m.t, m.expectedHeader, r.Header)
	return &http.Response{}, nil
}

//nolint:bodyclose // This is the test function and resp.Body is always nil in these tests
func TestMimirRoundTripper(t *testing.T) {
	testScopeOrg := "12345"
	canonicalHeaderXScopeOrgID := http.CanonicalHeaderKey(HeaderXScopeOrgID)
	mimirRT := newMimirRoundTripper(&testScopeOrg)

	t.Run("header is added", func(t *testing.T) {
		mimirRT.rt = newRoundTripMock(t, http.Header{canonicalHeaderXScopeOrgID: []string{testScopeOrg}})
		req := newTestRequest(nil)
		_, err := mimirRT.RoundTrip(req)
		require.NoError(t, err)
	})

	t.Run("all headers are added", func(t *testing.T) {
		mimirRT.rt = newRoundTripMock(t, http.Header{
			canonicalHeaderXScopeOrgID: []string{testScopeOrg},
			"Test":                     []string{"test"},
		})
		req := newTestRequest(http.Header{"Test": []string{"test"}})
		_, err := mimirRT.RoundTrip(req)
		require.NoError(t, err)
	})

	t.Run("header is not changed when 'X-Scope-OrgID' present", func(t *testing.T) {
		expectedOrgID := "123"
		expectedHeader := http.Header{HeaderXScopeOrgID: []string{expectedOrgID}}
		mimirRT.rt = newRoundTripMock(t, expectedHeader)
		req := newTestRequest(expectedHeader)
		_, err := mimirRT.RoundTrip(req)
		require.NoError(t, err)
	})

	t.Run("header is not changed when 'X-Scope-OrgId' present", func(t *testing.T) {
		expectedOrgID := "123"
		expectedHeader := http.Header{canonicalHeaderXScopeOrgID: []string{expectedOrgID}}
		mimirRT.rt = newRoundTripMock(t, expectedHeader)
		req := newTestRequest(expectedHeader)
		_, err := mimirRT.RoundTrip(req)
		require.NoError(t, err)
	})
}

func newRoundTripMock(t *testing.T, expectedHeaders http.Header) http.RoundTripper {
	return &mimirRoundTripper{
		rt: &mockRoundTrip{t, expectedHeaders},
	}
}

func newTestRequest(header http.Header) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "http://test.com", nil)
	req.Header = header
	return req
}
