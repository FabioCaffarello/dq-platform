// path: tools/lint/reachability_test.go

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- splitChannelRef ---

func TestSplitChannelRef(t *testing.T) {
	cases := []struct {
		ref       string
		wantType  string
		wantID    string
		wantValid bool
	}{
		{"slack:#dq-customer", "slack", "#dq-customer", true},
		{"email:oncall@example.com", "email", "oncall@example.com", true},
		{"pagerduty:P12345AB", "pagerduty", "P12345AB", true},
		{"slack:", "", "", false},  // empty id
		{":foo", "", "", false},    // empty type
		{"plainstring", "", "", false}, // no colon
		{"", "", "", false},        // empty
	}
	for _, tc := range cases {
		typ, id, ok := splitChannelRef(tc.ref)
		if ok != tc.wantValid {
			t.Errorf("splitChannelRef(%q): ok=%v; want %v", tc.ref, ok, tc.wantValid)
			continue
		}
		if !ok {
			continue
		}
		if typ != tc.wantType || id != tc.wantID {
			t.Errorf("splitChannelRef(%q) = (%q, %q); want (%q, %q)",
				tc.ref, typ, id, tc.wantType, tc.wantID)
		}
	}
}

// --- Slack adapter ---

func TestSlackAdapter_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("channel"); got != "dq-customer" {
			t.Errorf("Slack adapter sent channel=%q; want dq-customer (leading # should be stripped)", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q; want Bearer test-token", got)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok": true, "channel": {"id": "C123", "name": "dq-customer"}}`)
	}))
	defer srv.Close()

	a := NewSlackAdapter("test-token", srv.Client(), srv.URL)
	r := a.Check(context.Background(), "#dq-customer")
	if r.Status != ReachabilityPass {
		t.Errorf("Status = %q; want pass. Reason: %s", r.Status, r.Reason)
	}
}

func TestSlackAdapter_CredentialAbsent(t *testing.T) {
	a := NewSlackAdapter("", nil, "")
	r := a.Check(context.Background(), "#anywhere")
	if r.Status != ReachabilitySkipped {
		t.Errorf("Status = %q; want skipped", r.Status)
	}
	if !strings.Contains(r.Reason, "DQ_LINT_SLACK_TOKEN") {
		t.Errorf("Reason should mention DQ_LINT_SLACK_TOKEN; got %q", r.Reason)
	}
}

func TestSlackAdapter_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok": false, "error": "channel_not_found"}`)
	}))
	defer srv.Close()

	a := NewSlackAdapter("test-token", srv.Client(), srv.URL)
	r := a.Check(context.Background(), "#nope")
	if r.Status != ReachabilityFail {
		t.Errorf("Status = %q; want fail", r.Status)
	}
	if !strings.Contains(r.Reason, "channel_not_found") {
		t.Errorf("Reason should carry the Slack error code; got %q", r.Reason)
	}
}

func TestSlackAdapter_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := NewSlackAdapter("test-token", srv.Client(), srv.URL)
	r := a.Check(context.Background(), "#anywhere")
	if r.Status != ReachabilityFail {
		t.Errorf("Status = %q; want fail", r.Status)
	}
	if !strings.Contains(r.Reason, "500") {
		t.Errorf("Reason should mention HTTP 500; got %q", r.Reason)
	}
}

// --- Email adapter ---

type fakeMXResolver struct {
	records []*net.MX
	err     error
}

func (f *fakeMXResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	return f.records, f.err
}

func TestEmailAdapter_HappyPath(t *testing.T) {
	a := NewEmailAdapter(&fakeMXResolver{
		records: []*net.MX{{Host: "mx1.example.com.", Pref: 10}},
	})
	r := a.Check(context.Background(), "oncall@example.com")
	if r.Status != ReachabilityPass {
		t.Errorf("Status = %q; want pass. Reason: %s", r.Status, r.Reason)
	}
}

func TestEmailAdapter_NoMXRecords(t *testing.T) {
	a := NewEmailAdapter(&fakeMXResolver{records: nil})
	r := a.Check(context.Background(), "oncall@no-mx.example.com")
	if r.Status != ReachabilityFail {
		t.Errorf("Status = %q; want fail", r.Status)
	}
	if !strings.Contains(r.Reason, "no MX") {
		t.Errorf("Reason should mention 'no MX'; got %q", r.Reason)
	}
}

func TestEmailAdapter_LookupError(t *testing.T) {
	a := NewEmailAdapter(&fakeMXResolver{err: errors.New("dial: timeout")})
	r := a.Check(context.Background(), "oncall@example.com")
	if r.Status != ReachabilityFail {
		t.Errorf("Status = %q; want fail", r.Status)
	}
	if !strings.Contains(r.Reason, "timeout") {
		t.Errorf("Reason should carry the underlying error; got %q", r.Reason)
	}
}

func TestEmailAdapter_MalformedAddress(t *testing.T) {
	a := NewEmailAdapter(&fakeMXResolver{})
	cases := []string{"not-an-email", "@no-user.com", "no-domain@"}
	for _, addr := range cases {
		r := a.Check(context.Background(), addr)
		if r.Status != ReachabilityFail {
			t.Errorf("Status for %q = %q; want fail", addr, r.Status)
		}
	}
}

// --- PagerDuty adapter ---

func TestPagerDutyAdapter_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Token token=test-key" {
			t.Errorf("Authorization = %q; want 'Token token=test-key'", got)
		}
		if !strings.HasSuffix(r.URL.Path, "/services/P12345AB") {
			t.Errorf("path = %q; want suffix /services/P12345AB", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"service": {"id": "P12345AB"}}`)
	}))
	defer srv.Close()

	a := NewPagerDutyAdapter("test-key", srv.Client(), srv.URL)
	r := a.Check(context.Background(), "P12345AB")
	if r.Status != ReachabilityPass {
		t.Errorf("Status = %q; want pass. Reason: %s", r.Status, r.Reason)
	}
}

func TestPagerDutyAdapter_CredentialAbsent(t *testing.T) {
	a := NewPagerDutyAdapter("", nil, "")
	r := a.Check(context.Background(), "P12345AB")
	if r.Status != ReachabilitySkipped {
		t.Errorf("Status = %q; want skipped", r.Status)
	}
	if !strings.Contains(r.Reason, "DQ_LINT_PAGERDUTY_KEY") {
		t.Errorf("Reason should mention DQ_LINT_PAGERDUTY_KEY; got %q", r.Reason)
	}
}

func TestPagerDutyAdapter_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	a := NewPagerDutyAdapter("test-key", srv.Client(), srv.URL)
	r := a.Check(context.Background(), "PNOPE")
	if r.Status != ReachabilityFail {
		t.Errorf("Status = %q; want fail", r.Status)
	}
	if !strings.Contains(r.Reason, "404") {
		t.Errorf("Reason should mention HTTP 404; got %q", r.Reason)
	}
}

func TestPagerDutyAdapter_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	a := NewPagerDutyAdapter("bad-key", srv.Client(), srv.URL)
	r := a.Check(context.Background(), "P12345AB")
	if r.Status != ReachabilityFail {
		t.Errorf("Status = %q; want fail", r.Status)
	}
	if !strings.Contains(r.Reason, "auth") {
		t.Errorf("Reason should mention auth failure; got %q", r.Reason)
	}
}

// --- CheckChannelReachability walker ---

func TestCheckChannelReachability_HappyPath(t *testing.T) {
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"customer": {
				Channels: map[string][]string{
					"data_quality": {"slack:#dq-customer"},
					"operational":  {"email:oncall@example.com"},
				},
			},
			"orders": {
				Channels: map[string][]string{
					"data_quality": {"pagerduty:PSERVICE"},
				},
			},
		},
	}

	// Stub adapters: each returns a deterministic pass.
	reg := NewAdapterRegistry()
	reg.Register(&stubAdapter{typ: "slack", status: ReachabilityPass, reason: "stub-pass"})
	reg.Register(&stubAdapter{typ: "email", status: ReachabilityPass, reason: "stub-pass"})
	reg.Register(&stubAdapter{typ: "pagerduty", status: ReachabilityPass, reason: "stub-pass"})

	results := CheckChannelReachability(context.Background(), owners, reg)
	if len(results) != 3 {
		t.Fatalf("got %d results; want 3", len(results))
	}
	// Sorted: (customer, data_quality, slack:#dq-customer), (customer, operational, email:...), (orders, ...).
	if results[0].Entity != "customer" || results[0].Category != "data_quality" {
		t.Errorf("first result = %+v; want customer/data_quality", results[0])
	}
	if results[2].Entity != "orders" {
		t.Errorf("third result entity = %q; want orders", results[2].Entity)
	}
	for _, r := range results {
		if r.Status != ReachabilityPass {
			t.Errorf("result %+v: status = %q; want pass", r, r.Status)
		}
	}
}

func TestCheckChannelReachability_UnknownChannelType(t *testing.T) {
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"customer": {
				Channels: map[string][]string{
					"data_quality": {"webhook:https://example.com/hook"},
				},
			},
		},
	}
	reg := NewAdapterRegistry() // empty
	results := CheckChannelReachability(context.Background(), owners, reg)
	if len(results) != 1 {
		t.Fatalf("got %d results; want 1", len(results))
	}
	if results[0].Status != ReachabilitySkipped {
		t.Errorf("status = %q; want skipped", results[0].Status)
	}
	if !strings.Contains(results[0].Reason, "unknown channel type") {
		t.Errorf("reason should mention 'unknown channel type'; got %q", results[0].Reason)
	}
}

func TestCheckChannelReachability_MalformedRef(t *testing.T) {
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"customer": {
				Channels: map[string][]string{
					"data_quality": {"no-colon-ref"},
				},
			},
		},
	}
	reg := NewAdapterRegistry()
	reg.Register(&stubAdapter{typ: "slack", status: ReachabilityPass})
	results := CheckChannelReachability(context.Background(), owners, reg)
	if len(results) != 1 {
		t.Fatalf("got %d results; want 1", len(results))
	}
	if results[0].Status != ReachabilitySkipped {
		t.Errorf("status = %q; want skipped", results[0].Status)
	}
	if !strings.Contains(results[0].Reason, "malformed") {
		t.Errorf("reason should mention 'malformed'; got %q", results[0].Reason)
	}
}

func TestCheckChannelReachability_EmptyOwners(t *testing.T) {
	for _, owners := range []*Owners{nil, {Entities: map[string]OwnerEntity{}}} {
		results := CheckChannelReachability(context.Background(), owners, NewDefaultRegistry())
		if len(results) != 0 {
			t.Errorf("empty owners returned %d results; want 0", len(results))
		}
	}
}

func TestNewDefaultRegistry_RegistersThreeAdapters(t *testing.T) {
	reg := NewDefaultRegistry()
	for _, want := range []string{"slack", "email", "pagerduty"} {
		if _, ok := reg.byType[want]; !ok {
			t.Errorf("default registry missing %q adapter", want)
		}
	}
}

// --- helpers ---

type stubAdapter struct {
	typ    string
	status ReachabilityStatus
	reason string
}

func (s *stubAdapter) ChannelType() string { return s.typ }
func (s *stubAdapter) Check(_ context.Context, _ string) ReachabilityResult {
	return ReachabilityResult{Status: s.status, Reason: s.reason}
}
