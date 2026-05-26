// path: engine/internal/runner/runner_test.go

package runner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"dq-platform/engine/internal/alerts"
	"dq-platform/engine/internal/results"
)

// --- Compute tests (ADR-0002 CC1/CC2) ---

func TestCompute_HappyPath(t *testing.T) {
	id, err := Compute("rules-v1.0.0", "customer",
		time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC),
		results.TriggerScheduler)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(id) != 64 {
		t.Errorf("execution_id length = %d; want 64", len(id))
	}
	// Lowercase hex per ADR-0002 CC7.
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("execution_id contains non-lowercase-hex char %q in %q", c, id)
			break
		}
	}
}

func TestCompute_Determinism(t *testing.T) {
	inputs := func() (string, string, time.Time, time.Time, results.TriggerSource) {
		return "rules-v1.0.0", "customer",
			time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
			time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC),
			results.TriggerScheduler
	}
	id1, _ := Compute(inputs())
	id2, _ := Compute(inputs())
	if id1 != id2 {
		t.Errorf("Compute is non-deterministic: %q vs %q", id1, id2)
	}
}

func TestCompute_DifferentInputs_DifferentIDs(t *testing.T) {
	id1, _ := Compute("rules-v1.0.0", "customer",
		time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC),
		results.TriggerScheduler)
	id2, _ := Compute("rules-v1.0.0", "customer",
		time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC),
		results.TriggerManual) // different trigger source
	if id1 == id2 {
		t.Errorf("Compute should differ on different trigger_source")
	}
}

func TestCompute_EntityWithPipe_Rejected(t *testing.T) {
	_, err := Compute("rules-v1.0.0", "cust|omer",
		time.Now(), time.Now().Add(time.Hour), results.TriggerScheduler)
	if !errors.Is(err, ErrPipeCharacterForbidden) {
		t.Errorf("err = %v; want ErrPipeCharacterForbidden", err)
	}
}

func TestCompute_RulesetVersionWithPipe_Rejected(t *testing.T) {
	_, err := Compute("rules-v|0", "customer",
		time.Now(), time.Now().Add(time.Hour), results.TriggerScheduler)
	if !errors.Is(err, ErrPipeCharacterForbidden) {
		t.Errorf("err = %v; want ErrPipeCharacterForbidden", err)
	}
}

func TestCompute_TruncatesSubSecondPrecision(t *testing.T) {
	// Two timestamps differing only in sub-second component
	// must produce the same execution_id (ADR-0002 CC2: second
	// precision; fractional dropped).
	t0 := time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 5, 21, 14, 0, 0, 999_999_999, time.UTC) // same second
	id0, _ := Compute("rules-v1.0.0", "customer", t0, t0.Add(time.Hour), results.TriggerScheduler)
	id1, _ := Compute("rules-v1.0.0", "customer", t1, t1.Add(time.Hour), results.TriggerScheduler)
	if id0 != id1 {
		t.Errorf("Compute does not truncate sub-second precision: %q vs %q", id0, id1)
	}
}

// --- MapStatus tests (ADR-0004 CC2) ---

func TestMapStatus_AllPass(t *testing.T) {
	s, err := MapStatus([]results.CheckResult{results.ResultPass, results.ResultPass})
	if err != nil || s != results.StatusSuccess {
		t.Errorf("MapStatus(all pass) = (%q, %v); want (success, nil)", s, err)
	}
}

func TestMapStatus_AllError(t *testing.T) {
	s, err := MapStatus([]results.CheckResult{results.ResultError, results.ResultError})
	if err != nil || s != results.StatusError {
		t.Errorf("MapStatus(all error) = (%q, %v); want (error, nil)", s, err)
	}
}

func TestMapStatus_MixedPassFail(t *testing.T) {
	s, err := MapStatus([]results.CheckResult{results.ResultPass, results.ResultFail})
	if err != nil || s != results.StatusFailed {
		t.Errorf("MapStatus(mixed) = (%q, %v); want (failed, nil)", s, err)
	}
}

func TestMapStatus_MixedPassError(t *testing.T) {
	// ADR-0004 CC3 promotion rule: one error among passes → failed,
	// not error.
	s, err := MapStatus([]results.CheckResult{results.ResultPass, results.ResultError})
	if err != nil || s != results.StatusFailed {
		t.Errorf("MapStatus(pass+error) = (%q, %v); want (failed, nil)", s, err)
	}
}

func TestMapStatus_Degraded_CountsAsMixed(t *testing.T) {
	s, err := MapStatus([]results.CheckResult{results.ResultDegraded, results.ResultPass})
	if err != nil || s != results.StatusFailed {
		t.Errorf("MapStatus(degraded+pass) = (%q, %v); want (failed, nil)", s, err)
	}
}

func TestMapStatus_Empty_ReturnsError(t *testing.T) {
	_, err := MapStatus(nil)
	if !errors.Is(err, ErrEmptyResultMultiset) {
		t.Errorf("MapStatus(empty) = %v; want ErrEmptyResultMultiset", err)
	}
}

// --- Run tests ---

// inMemStore is a minimal Store impl for runner tests. The
// existing results package has a mockStore in its own
// _test.go, but that's not exported; this is a local copy
// for the runner's tests.
type inMemStore struct {
	executions []results.ExecutionRow
	checks     []results.CheckResultRow
}

func (s *inMemStore) EnsureSchema(_ context.Context) error { return nil }

func (s *inMemStore) WriteExecutionRow(_ context.Context, row results.ExecutionRow) error {
	s.executions = append(s.executions, row)
	return nil
}

func (s *inMemStore) WriteCheckResultRow(_ context.Context, row results.CheckResultRow) error {
	s.checks = append(s.checks, row)
	return nil
}

func (s *inMemStore) QueryCurrentExecution(_ context.Context, executionID string) (*results.ExecutionRow, error) {
	var latest *results.ExecutionRow
	for i := range s.executions {
		r := s.executions[i]
		if r.ExecutionID != executionID {
			continue
		}
		if latest == nil || r.RecordedAt.After(latest.RecordedAt) {
			cp := r
			latest = &cp
		}
	}
	if latest == nil {
		return nil, results.ErrExecutionNotFound
	}
	return latest, nil
}

func (s *inMemStore) ListRunningOlderThan(_ context.Context, _ time.Time) ([]results.ExecutionRow, error) {
	return nil, nil
}

func (s *inMemStore) LatestExecutionPerEntityCheck(_ context.Context, _ time.Time) ([]results.LatestExecutionRow, error) {
	return nil, nil
}

func newTestRunner(t *testing.T, store results.Store, evaluator CheckEvaluator) *Runner {
	t.Helper()
	r, err := New(Config{
		Store:          store,
		Evaluator:      evaluator,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func sampleTrigger() TriggerRequest {
	return TriggerRequest{
		Entity:        "customer",
		WindowStart:   time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC),
		TriggerSource: results.TriggerScheduler,
		Checks: []CheckSpec{
			{CheckID: "row_count_positive", Kind: "stub"},
		},
	}
}

func TestNew_RequiresStore(t *testing.T) {
	_, err := New(Config{EngineVersion: "0.1.0", RulesetVersion: "rules-v1.0.0"})
	if err == nil {
		t.Fatalf("New without Store returned nil; want error")
	}
}

func TestNew_RequiresEngineVersion(t *testing.T) {
	_, err := New(Config{Store: &inMemStore{}, RulesetVersion: "rules-v1.0.0"})
	if err == nil {
		t.Fatalf("New without EngineVersion returned nil; want error")
	}
}

func TestNew_RequiresValidSemver(t *testing.T) {
	_, err := New(Config{Store: &inMemStore{}, EngineVersion: "not-semver", RulesetVersion: "rules-v1.0.0"})
	if err == nil {
		t.Fatalf("New with non-semver EngineVersion returned nil; want error")
	}
}

func TestRun_HappyPath_AllPass(t *testing.T) {
	store := &inMemStore{}
	r := newTestRunner(t, store, NoopEvaluator{})

	terminal, err := r.Run(context.Background(), sampleTrigger())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusSuccess {
		t.Errorf("terminal status = %q; want success", terminal.Status)
	}

	// Two execution rows expected: running + terminal.
	if len(store.executions) != 2 {
		t.Fatalf("execution rows = %d; want 2 (running + terminal)", len(store.executions))
	}
	if store.executions[0].Status != results.StatusRunning {
		t.Errorf("first row status = %q; want running", store.executions[0].Status)
	}
	if store.executions[1].Status != results.StatusSuccess {
		t.Errorf("second row status = %q; want success", store.executions[1].Status)
	}
	// One check row expected.
	if len(store.checks) != 1 {
		t.Errorf("check rows = %d; want 1", len(store.checks))
	}
}

func TestRun_PrecheckAbsent_WritesErrorRowNoChecks(t *testing.T) {
	store := &inMemStore{}
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		Precheck:       fixedPrecheck{present: false},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	terminal, err := r.Run(context.Background(), sampleTrigger())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusError {
		t.Errorf("status = %q; want error (pre-check absent)", terminal.Status)
	}
	if len(store.executions) != 1 {
		t.Errorf("execution rows = %d; want 1 (terminal error, no running)", len(store.executions))
	}
	if len(store.checks) != 0 {
		t.Errorf("check rows = %d; want 0 (pre-check error path)", len(store.checks))
	}
}

func TestRun_PrecheckOperationalFailure(t *testing.T) {
	store := &inMemStore{}
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		Precheck:       fixedPrecheck{err: errors.New("metadata API unreachable")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	terminal, err := r.Run(context.Background(), sampleTrigger())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusError {
		t.Errorf("status = %q; want error", terminal.Status)
	}
	if terminal.ErrorSummary == nil || !strings.Contains(*terminal.ErrorSummary, "metadata API unreachable") {
		t.Errorf("ErrorSummary = %v; want it to mention the precheck error", terminal.ErrorSummary)
	}
}

func TestRun_MixedResults_TerminalFailed(t *testing.T) {
	store := &inMemStore{}
	trigger := sampleTrigger()
	trigger.Checks = []CheckSpec{
		{CheckID: "a", Kind: "stub"},
		{CheckID: "b", Kind: "stub"},
	}
	evaluator := PerCheckEvaluator{
		Results: map[string]results.CheckResult{
			"a": results.ResultPass,
			"b": results.ResultFail,
		},
	}
	r := newTestRunner(t, store, evaluator)
	terminal, err := r.Run(context.Background(), trigger)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusFailed {
		t.Errorf("status = %q; want failed", terminal.Status)
	}
	// ADR-0003 CC3: terminal row with status=failed must carry an
	// error_summary; should mention how many checks did not pass.
	if terminal.ErrorSummary == nil {
		t.Fatalf("ErrorSummary is nil; want set per ADR-0003 CC3")
	}
	if !strings.Contains(*terminal.ErrorSummary, "1 of 2") {
		t.Errorf("ErrorSummary = %q; want it to say 1 of 2", *terminal.ErrorSummary)
	}
}

func TestRun_AllError_TerminalErrorWithSummary(t *testing.T) {
	store := &inMemStore{}
	trigger := sampleTrigger()
	trigger.Checks = []CheckSpec{
		{CheckID: "a", Kind: "stub"},
		{CheckID: "b", Kind: "stub"},
	}
	evaluator := FixedResultEvaluator{Result: results.ResultError}
	r := newTestRunner(t, store, evaluator)
	terminal, err := r.Run(context.Background(), trigger)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusError {
		t.Errorf("status = %q; want error (all checks errored)", terminal.Status)
	}
	// ADR-0003 CC3: terminal row with status=error must carry an
	// error_summary.
	if terminal.ErrorSummary == nil {
		t.Fatalf("ErrorSummary is nil; want set per ADR-0003 CC3")
	}
	if !strings.Contains(*terminal.ErrorSummary, "all 2 checks errored") {
		t.Errorf("ErrorSummary = %q; want it to say 'all 2 checks errored'", *terminal.ErrorSummary)
	}
}

func TestRun_AllPass_TerminalSuccess_NoSummary(t *testing.T) {
	// Companion to the above: status=success has no ErrorSummary.
	store := &inMemStore{}
	r := newTestRunner(t, store, NoopEvaluator{})
	terminal, err := r.Run(context.Background(), sampleTrigger())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusSuccess {
		t.Errorf("status = %q; want success", terminal.Status)
	}
	if terminal.ErrorSummary != nil {
		t.Errorf("ErrorSummary = %q; want nil for success", *terminal.ErrorSummary)
	}
}

func TestRun_OperatorRerun_RequiresSupersedes(t *testing.T) {
	store := &inMemStore{}
	r := newTestRunner(t, store, NoopEvaluator{})
	trigger := sampleTrigger()
	trigger.TriggerSource = results.TriggerOperatorRerun // no SupersedesExecutionID

	_, err := r.Run(context.Background(), trigger)
	if err == nil {
		t.Fatalf("Run(operator-rerun without supersedes) returned nil; want error")
	}
	if !strings.Contains(err.Error(), "SupersedesExecutionID") {
		t.Errorf("err %q should mention SupersedesExecutionID", err)
	}
}

func TestRun_SchedulerWithSupersedes_Rejected(t *testing.T) {
	store := &inMemStore{}
	r := newTestRunner(t, store, NoopEvaluator{})
	trigger := sampleTrigger()
	supers := "deadbeef"
	trigger.SupersedesExecutionID = &supers

	_, err := r.Run(context.Background(), trigger)
	if err == nil {
		t.Fatalf("Run(scheduler with supersedes) returned nil; want error")
	}
}

func TestRun_ExecutionIDStableAcrossRows(t *testing.T) {
	store := &inMemStore{}
	r := newTestRunner(t, store, NoopEvaluator{})
	terminal, err := r.Run(context.Background(), sampleTrigger())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Every execution row + every check row carries the same id.
	for i, row := range store.executions {
		if row.ExecutionID != terminal.ExecutionID {
			t.Errorf("execution row %d ID = %q; want %q", i, row.ExecutionID, terminal.ExecutionID)
		}
	}
	for i, row := range store.checks {
		if row.ExecutionID != terminal.ExecutionID {
			t.Errorf("check row %d ID = %q; want %q", i, row.ExecutionID, terminal.ExecutionID)
		}
	}
}

func TestRun_AttemptIDStableWithinRun(t *testing.T) {
	store := &inMemStore{}
	r := newTestRunner(t, store, NoopEvaluator{})
	if _, err := r.Run(context.Background(), sampleTrigger()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for i, row := range store.executions {
		if row.AttemptID != "att-1" {
			t.Errorf("execution row %d AttemptID = %q; want att-1", i, row.AttemptID)
		}
	}
}

func TestRun_EvaluatorError_BecomesResultError(t *testing.T) {
	// ADR-0004 CC1: evaluator errors map to ResultError, not
	// Run-level failures. The runner records the row and
	// continues with siblings.
	store := &inMemStore{}
	trigger := sampleTrigger()
	trigger.Checks = []CheckSpec{
		{CheckID: "a", Kind: "stub"},
		{CheckID: "b", Kind: "stub"},
	}
	evaluator := perCheckEvaluatorWithError{
		failOn: "a",
	}
	r := newTestRunner(t, store, evaluator)
	terminal, err := r.Run(context.Background(), trigger)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusFailed {
		t.Errorf("terminal status = %q; want failed (one ResultError + one pass = mixed)", terminal.Status)
	}
	if len(store.checks) != 2 {
		t.Fatalf("check rows = %d; want 2 (evaluator error still produces a row)", len(store.checks))
	}
	// First check should be result=error, second result=pass.
	if store.checks[0].Result != results.ResultError {
		t.Errorf("check[0].Result = %q; want error", store.checks[0].Result)
	}
	if store.checks[1].Result != results.ResultPass {
		t.Errorf("check[1].Result = %q; want pass", store.checks[1].Result)
	}
}

func TestRun_ZeroChecks_TerminalErrorWithSummary(t *testing.T) {
	store := &inMemStore{}
	r := newTestRunner(t, store, NoopEvaluator{})
	trigger := sampleTrigger()
	trigger.Checks = nil // no checks

	terminal, err := r.Run(context.Background(), trigger)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusError {
		t.Errorf("status = %q; want error (zero checks)", terminal.Status)
	}
	if terminal.ErrorSummary == nil || !strings.Contains(*terminal.ErrorSummary, "zero checks") {
		t.Errorf("ErrorSummary = %v; want 'zero checks' note", terminal.ErrorSummary)
	}
}

// --- test helpers ---

type fixedPrecheck struct {
	present bool
	err     error
}

func (p fixedPrecheck) SourceExists(_ context.Context, _ string) (bool, error) {
	return p.present, p.err
}

type perCheckEvaluatorWithError struct {
	failOn string
}

func (e perCheckEvaluatorWithError) Evaluate(_ context.Context, spec CheckSpec, _ TriggerRequest) (Evaluation, error) {
	if spec.CheckID == e.failOn {
		return Evaluation{}, errors.New("synthetic evaluator failure")
	}
	return Evaluation{Result: results.ResultPass}, nil
}

// --- publish-hook tests (ADR-0006 CC4 / CC7) ---

// capturePublisher records every Event handed to Publish. Used
// by the runner publish-hook tests to assert what was emitted.
// Safe for concurrent use; the runner publishes from one
// goroutine per Run today but the deduper concurrency contract
// already exercises multi-goroutine emission.
type capturePublisher struct {
	mu     sync.Mutex
	events []alerts.Event
	err    error // if non-nil, Publish returns this and still records
}

func (p *capturePublisher) Publish(_ context.Context, e alerts.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
	return p.err
}

func (p *capturePublisher) snapshot() []alerts.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]alerts.Event, len(p.events))
	copy(out, p.events)
	return out
}

func TestRun_HappyPath_NoAlerts(t *testing.T) {
	// ADR-0006 CC7: all checks passing → status=success →
	// neither check-level nor execution-level alert.
	store := &inMemStore{}
	pub := &capturePublisher{}
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		Publisher:      pub,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := r.Run(context.Background(), sampleTrigger()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := pub.snapshot(); len(got) != 0 {
		t.Errorf("happy-path emitted %d alert(s); want 0: %+v", len(got), got)
	}
}

func TestRun_OneFail_EmitsCheckAndExecutionAlerts(t *testing.T) {
	// ADR-0006 CC7: one failing check → one data_quality
	// check-level alert + one operational execution-level alert
	// (status=failed).
	store := &inMemStore{}
	pub := &capturePublisher{}
	trigger := sampleTrigger()
	trigger.Checks = []CheckSpec{
		{CheckID: "a", Kind: "stub"},
		{CheckID: "b", Kind: "stub"},
	}
	evaluator := PerCheckEvaluator{
		Results: map[string]results.CheckResult{
			"a": results.ResultPass,
			"b": results.ResultFail,
		},
	}
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		Evaluator:      evaluator,
		Publisher:      pub,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := r.Run(context.Background(), trigger); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events := pub.snapshot()
	if len(events) != 2 {
		t.Fatalf("alerts emitted = %d; want 2 (one check-level + one execution-level): %+v", len(events), events)
	}
	// First emit: check-level for the failing check; second emit:
	// execution-level. Pass result on "a" must not emit.
	checkLevel, executionLevel := events[0], events[1]
	if checkLevel.CheckID == nil || *checkLevel.CheckID != "b" {
		t.Errorf("first event CheckID = %v; want b", checkLevel.CheckID)
	}
	if checkLevel.Category != alerts.CategoryDataQuality {
		t.Errorf("check-level Category = %q; want data_quality", checkLevel.Category)
	}
	if checkLevel.Result == nil || *checkLevel.Result != results.ResultFail {
		t.Errorf("check-level Result = %v; want fail", checkLevel.Result)
	}
	if executionLevel.CheckID != nil {
		t.Errorf("execution-level event must not carry CheckID; got %v", *executionLevel.CheckID)
	}
	if executionLevel.Category != alerts.CategoryOperational {
		t.Errorf("execution-level Category = %q; want operational", executionLevel.Category)
	}
	if executionLevel.Status == nil || *executionLevel.Status != results.StatusFailed {
		t.Errorf("execution-level Status = %v; want failed", executionLevel.Status)
	}
	if executionLevel.ErrorSummary == nil {
		t.Errorf("execution-level ErrorSummary is nil; want set per ADR-0003 CC3")
	}
}

func TestRun_AllError_EmitsCheckAndExecutionOperationalAlerts(t *testing.T) {
	// ADR-0006 CC7: check result=error maps to operational, not
	// data_quality. Two errored checks → two operational
	// check-level alerts + one operational execution-level alert
	// (status=error).
	store := &inMemStore{}
	pub := &capturePublisher{}
	trigger := sampleTrigger()
	trigger.Checks = []CheckSpec{
		{CheckID: "a", Kind: "stub"},
		{CheckID: "b", Kind: "stub"},
	}
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		Evaluator:      FixedResultEvaluator{Result: results.ResultError},
		Publisher:      pub,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := r.Run(context.Background(), trigger); err != nil {
		t.Fatalf("Run: %v", err)
	}
	events := pub.snapshot()
	if len(events) != 3 {
		t.Fatalf("alerts emitted = %d; want 3 (2 check-level + 1 execution-level): %+v", len(events), events)
	}
	for i := 0; i < 2; i++ {
		if events[i].Category != alerts.CategoryOperational {
			t.Errorf("check-level event %d Category = %q; want operational (ResultError)", i, events[i].Category)
		}
	}
	if events[2].Category != alerts.CategoryOperational {
		t.Errorf("execution-level Category = %q; want operational", events[2].Category)
	}
}

func TestRun_PrecheckAbsent_EmitsOperationalAlert(t *testing.T) {
	// ADR-0006 CC7: precheck absent → terminal status=error →
	// one operational execution-level alert, no check-level
	// alerts.
	store := &inMemStore{}
	pub := &capturePublisher{}
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		Precheck:       fixedPrecheck{present: false},
		Publisher:      pub,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := r.Run(context.Background(), sampleTrigger()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	events := pub.snapshot()
	if len(events) != 1 {
		t.Fatalf("alerts emitted = %d; want exactly 1 (operational execution-level): %+v", len(events), events)
	}
	e := events[0]
	if e.Category != alerts.CategoryOperational {
		t.Errorf("Category = %q; want operational", e.Category)
	}
	if e.EventSource != alerts.SourceRunner {
		t.Errorf("EventSource = %q; want runner", e.EventSource)
	}
	if e.Status == nil || *e.Status != results.StatusError {
		t.Errorf("Status = %v; want error", e.Status)
	}
	if e.ErrorSummary == nil || !strings.Contains(*e.ErrorSummary, "source not present") {
		t.Errorf("ErrorSummary = %v; want it to mention 'source not present'", e.ErrorSummary)
	}
	if e.CheckID != nil {
		t.Errorf("execution-level event must not carry CheckID; got %v", *e.CheckID)
	}
}

func TestRun_PublishErrorDoesNotFailRun(t *testing.T) {
	// ADR-0006 CC4 + CC5: alerting is best-effort. Publish
	// failures must be logged but not propagated; the execution
	// finalization is durable independent of the alert.
	store := &inMemStore{}
	pub := &capturePublisher{err: errors.New("synthetic publish failure")}
	trigger := sampleTrigger()
	trigger.Checks = []CheckSpec{{CheckID: "b", Kind: "stub"}}
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		Evaluator:      FixedResultEvaluator{Result: results.ResultFail},
		Publisher:      pub,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	terminal, err := r.Run(context.Background(), trigger)
	if err != nil {
		t.Fatalf("Run must not return publish failures; got %v", err)
	}
	if terminal.Status != results.StatusFailed {
		t.Errorf("terminal status = %q; want failed", terminal.Status)
	}
	// Both events were attempted even though Publish errored.
	if got := pub.snapshot(); len(got) != 2 {
		t.Errorf("publish attempts = %d; want 2", len(got))
	}
}

func TestRun_DedupSuppression_NoDoubleEmit(t *testing.T) {
	// ADR-0006 CC5: the engine-side deduper suppresses literal
	// duplicates within an attempt. Run with two trigger.Checks
	// sharing the same CheckID + the same result; the runner
	// should publish only one check-level event.
	store := &inMemStore{}
	pub := &capturePublisher{}
	trigger := sampleTrigger()
	trigger.Checks = []CheckSpec{
		{CheckID: "duplicate", Kind: "stub"},
		{CheckID: "duplicate", Kind: "stub"},
	}
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-1" },
		Now:            func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		Evaluator:      FixedResultEvaluator{Result: results.ResultFail},
		Publisher:      pub,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := r.Run(context.Background(), trigger); err != nil {
		t.Fatalf("Run: %v", err)
	}
	events := pub.snapshot()
	// Expected: 1 check-level (the duplicate emit is suppressed) +
	// 1 execution-level = 2 total.
	if len(events) != 2 {
		t.Fatalf("alerts emitted = %d; want 2 (1 check after dedup + 1 execution): %+v", len(events), events)
	}
	if events[0].CheckID == nil || *events[0].CheckID != "duplicate" {
		t.Errorf("first event CheckID = %v; want duplicate", events[0].CheckID)
	}
	if events[1].CheckID != nil {
		t.Errorf("second event must be execution-level (no CheckID); got %v", *events[1].CheckID)
	}
}
