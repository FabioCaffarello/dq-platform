// path: tools/lint/reachability.go

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// ReachabilityStatus is the closed enum of per-channel outcomes
// from the reachability check per ADR-0047 §"Failure semantics".
type ReachabilityStatus string

const (
	// ReachabilityPass — adapter confirmed the channel exists.
	ReachabilityPass ReachabilityStatus = "pass"

	// ReachabilityFail — adapter ran but the channel was not
	// reachable (HTTP 404, no MX records, etc.). Logged as
	// warning; exit code unchanged.
	ReachabilityFail ReachabilityStatus = "fail"

	// ReachabilitySkipped — adapter declined to run (credential
	// absent, unknown channel type, etc.). Logged as warning;
	// exit code unchanged.
	ReachabilitySkipped ReachabilityStatus = "skipped"
)

// ReachabilityResult is one per-channel outcome surfaced by
// CheckChannelReachability. The entity/category fields locate
// the channel in `_owners.yaml`; the channel field is the
// `<type>:<id>` literal; status + reason name the outcome.
type ReachabilityResult struct {
	Entity   string
	Category string
	Channel  string
	Status   ReachabilityStatus
	Reason   string
}

// adapter is the interface every per-channel-type reachability
// implementation satisfies. The Check method is non-mutating per
// ADR-0047 §"Adapters are never mutating": each impl reads;
// none post test messages.
type adapter interface {
	// ChannelType returns the lowercase prefix this adapter
	// handles (e.g., "slack", "email", "pagerduty"). The
	// registry dispatches on this value.
	ChannelType() string

	// Check resolves the channel id and returns a result. The
	// id is the substring after the colon in the
	// `<type>:<id>` reference. Implementations must NOT
	// mutate substrate state.
	Check(ctx context.Context, id string) ReachabilityResult
}

// AdapterRegistry maps channel type prefix → adapter. Construct
// via NewDefaultRegistry for the production set or by hand for
// tests that inject mocks.
type AdapterRegistry struct {
	byType map[string]adapter
}

// NewAdapterRegistry returns an empty registry. Callers Register
// each adapter explicitly. The empty registry produces
// `skipped — unknown channel type` for every channel.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{byType: map[string]adapter{}}
}

// Register installs an adapter under its declared ChannelType.
// Subsequent Register calls for the same type overwrite the
// prior entry.
func (r *AdapterRegistry) Register(a adapter) {
	r.byType[a.ChannelType()] = a
}

// NewDefaultRegistry wires the production adapter set per
// ADR-0047 §"Per-substrate adapter model": Slack, email,
// PagerDuty. Each adapter reads its credentials from the
// matching DQ_LINT_* env var (or skips cleanly when absent).
func NewDefaultRegistry() *AdapterRegistry {
	r := NewAdapterRegistry()
	r.Register(NewSlackAdapter(os.Getenv("DQ_LINT_SLACK_TOKEN"), nil, ""))
	r.Register(NewEmailAdapter(nil))
	r.Register(NewPagerDutyAdapter(os.Getenv("DQ_LINT_PAGERDUTY_KEY"), nil, ""))
	return r
}

// CheckChannelReachability walks every entity's per-category
// channel list and dispatches each `<type>:<id>` reference to
// the matching adapter. Returns a sorted (by entity, category,
// channel) list of results — sort order makes the operator-
// facing stderr output deterministic, easing PR diff review.
//
// Empty owners (missing file; v1 dispatcher empty) returns
// an empty slice; the linter's existing CC9 coverage handles
// the missing-owners case.
func CheckChannelReachability(ctx context.Context, owners *Owners, registry *AdapterRegistry) []ReachabilityResult {
	results := []ReachabilityResult{}
	if owners == nil || len(owners.Entities) == 0 {
		return results
	}

	names := make([]string, 0, len(owners.Entities))
	for n := range owners.Entities {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, entityName := range names {
		ent := owners.Entities[entityName]
		categories := make([]string, 0, len(ent.Channels))
		for c := range ent.Channels {
			categories = append(categories, c)
		}
		sort.Strings(categories)

		for _, cat := range categories {
			for _, ref := range ent.Channels[cat] {
				chanType, id, ok := splitChannelRef(ref)
				if !ok {
					results = append(results, ReachabilityResult{
						Entity:   entityName,
						Category: cat,
						Channel:  ref,
						Status:   ReachabilitySkipped,
						Reason:   "malformed channel reference (expected `<type>:<id>`)",
					})
					continue
				}
				a, ok := registry.byType[chanType]
				if !ok {
					results = append(results, ReachabilityResult{
						Entity:   entityName,
						Category: cat,
						Channel:  ref,
						Status:   ReachabilitySkipped,
						Reason:   fmt.Sprintf("unknown channel type %q (no adapter registered)", chanType),
					})
					continue
				}
				r := a.Check(ctx, id)
				r.Entity = entityName
				r.Category = cat
				r.Channel = ref
				results = append(results, r)
			}
		}
	}
	return results
}

// splitChannelRef parses `<type>:<id>` into its parts. The
// `<type>` is lowercase per ADR-0006's `channels` regex; the
// `<id>` is everything after the first colon. Returns ok=false
// when the colon is absent.
func splitChannelRef(ref string) (string, string, bool) {
	idx := strings.IndexByte(ref, ':')
	if idx <= 0 || idx == len(ref)-1 {
		return "", "", false
	}
	return ref[:idx], ref[idx+1:], true
}

// --- Slack adapter ---

// SlackAdapter checks `slack:#<channel-name>` references via
// the Slack `conversations.info` API. The API call requires a
// bot token with `channels:read` scope. When the token is
// empty, the adapter skips with `credential absent`.
type SlackAdapter struct {
	token      string
	client     *http.Client
	endpoint   string // override for tests; defaults to Slack's prod endpoint
}

// NewSlackAdapter constructs the Slack adapter. token is the
// DQ_LINT_SLACK_TOKEN value (empty disables); client overrides
// the HTTP client (nil = http.DefaultClient with a 5s timeout);
// endpointOverride overrides the API base URL (empty = Slack
// prod). Tests inject all three.
func NewSlackAdapter(token string, client *http.Client, endpointOverride string) *SlackAdapter {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	endpoint := endpointOverride
	if endpoint == "" {
		endpoint = "https://slack.com/api"
	}
	return &SlackAdapter{token: token, client: client, endpoint: endpoint}
}

// ChannelType returns "slack".
func (s *SlackAdapter) ChannelType() string { return "slack" }

// Check performs a non-mutating Slack API call against
// conversations.info. The id is the Slack channel name; the
// leading `#` is stripped if present.
func (s *SlackAdapter) Check(ctx context.Context, id string) ReachabilityResult {
	if s.token == "" {
		return ReachabilityResult{
			Status: ReachabilitySkipped,
			Reason: "credential absent (DQ_LINT_SLACK_TOKEN unset)",
		}
	}

	channelName := strings.TrimPrefix(id, "#")
	u := s.endpoint + "/conversations.info?channel=" + url.QueryEscape(channelName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("build request: %v", err),
		}
	}
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("http: %v", err),
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("Slack API returned HTTP %d", resp.StatusCode),
		}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("read body: %v", err),
		}
	}
	var payload struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("parse response: %v", err),
		}
	}
	if !payload.OK {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("Slack API error: %s", payload.Error),
		}
	}
	return ReachabilityResult{
		Status: ReachabilityPass,
		Reason: "Slack channel resolved",
	}
}

// --- Email adapter ---

// MXResolver is the per-test injection point for DNS MX
// lookups. Production uses net.DefaultResolver; tests supply a
// mock implementation. The signature mirrors
// (*net.Resolver).LookupMX so the production wrapper is a thin
// passthrough.
type MXResolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}

// defaultMXResolver adapts *net.Resolver to the MXResolver
// interface.
type defaultMXResolver struct{}

func (defaultMXResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return net.DefaultResolver.LookupMX(ctx, name)
}

// EmailAdapter validates `email:<address>` references by
// performing a DNS MX lookup on the address's domain. No
// message is sent; no credential is required.
type EmailAdapter struct {
	resolver MXResolver
}

// NewEmailAdapter constructs the email adapter. resolver=nil
// uses the default net.Resolver.
func NewEmailAdapter(resolver MXResolver) *EmailAdapter {
	if resolver == nil {
		resolver = defaultMXResolver{}
	}
	return &EmailAdapter{resolver: resolver}
}

// ChannelType returns "email".
func (e *EmailAdapter) ChannelType() string { return "email" }

// Check looks up MX records for the email's domain. One or more
// MX records → pass; zero or error → fail.
func (e *EmailAdapter) Check(ctx context.Context, id string) ReachabilityResult {
	atIdx := strings.LastIndexByte(id, '@')
	if atIdx <= 0 || atIdx == len(id)-1 {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("email address %q is malformed (expected user@domain)", id),
		}
	}
	domain := id[atIdx+1:]
	mxs, err := e.resolver.LookupMX(ctx, domain)
	if err != nil {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("MX lookup for %s: %v", domain, err),
		}
	}
	if len(mxs) == 0 {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("no MX records for %s", domain),
		}
	}
	return ReachabilityResult{
		Status: ReachabilityPass,
		Reason: fmt.Sprintf("%d MX record(s) for %s", len(mxs), domain),
	}
}

// --- PagerDuty adapter ---

// PagerDutyAdapter checks `pagerduty:<service-id>` references
// via the PagerDuty `/services/{id}` API. Requires an API key
// (DQ_LINT_PAGERDUTY_KEY) with read access to the services in
// scope.
type PagerDutyAdapter struct {
	apiKey   string
	client   *http.Client
	endpoint string // override for tests
}

// NewPagerDutyAdapter constructs the PagerDuty adapter.
func NewPagerDutyAdapter(apiKey string, client *http.Client, endpointOverride string) *PagerDutyAdapter {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	endpoint := endpointOverride
	if endpoint == "" {
		endpoint = "https://api.pagerduty.com"
	}
	return &PagerDutyAdapter{apiKey: apiKey, client: client, endpoint: endpoint}
}

// ChannelType returns "pagerduty".
func (p *PagerDutyAdapter) ChannelType() string { return "pagerduty" }

// Check performs a non-mutating PagerDuty API call against
// `/services/{id}`. HTTP 200 → pass; HTTP 404 → fail; HTTP 401
// → fail with credential message.
func (p *PagerDutyAdapter) Check(ctx context.Context, id string) ReachabilityResult {
	if p.apiKey == "" {
		return ReachabilityResult{
			Status: ReachabilitySkipped,
			Reason: "credential absent (DQ_LINT_PAGERDUTY_KEY unset)",
		}
	}
	u := p.endpoint + "/services/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("build request: %v", err),
		}
	}
	req.Header.Set("Authorization", "Token token="+p.apiKey)
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	resp, err := p.client.Do(req)
	if err != nil {
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("http: %v", err),
		}
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return ReachabilityResult{
			Status: ReachabilityPass,
			Reason: "PagerDuty service resolved",
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("PagerDuty auth failed (HTTP %d) — check DQ_LINT_PAGERDUTY_KEY scope", resp.StatusCode),
		}
	default:
		return ReachabilityResult{
			Status: ReachabilityFail,
			Reason: fmt.Sprintf("PagerDuty API returned HTTP %d", resp.StatusCode),
		}
	}
}
