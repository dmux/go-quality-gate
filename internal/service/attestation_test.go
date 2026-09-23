package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dmux/go-quality-gate/internal/domain"
)

type fakeCommit struct {
	tree    string
	message string
	config  []byte
}

// fakeGit is an in-memory CommitRepository and AttestationStateRepository.
type fakeGit struct {
	indexTree   string
	commits     map[string]fakeCommit
	order       []string
	attestation []byte
}

func (f *fakeGit) WriteTree() (string, error)               { return f.indexTree, nil }
func (f *fakeGit) CommitTree(rev string) (string, error)    { return f.commits[rev].tree, nil }
func (f *fakeGit) CommitMessage(rev string) (string, error) { return f.commits[rev].message, nil }
func (f *fakeGit) ShowFile(rev, _ string) ([]byte, error)   { return f.commits[rev].config, nil }
func (f *fakeGit) RevList(string) ([]string, error)         { return f.order, nil }
func (f *fakeGit) LoadAttestation() ([]byte, error)         { return f.attestation, nil }
func (f *fakeGit) DeleteAttestation() error                 { f.attestation = nil; return nil }

func (f *fakeGit) SaveAttestation(content []byte) error {
	f.attestation = content
	return nil
}

func (f *fakeGit) AddTrailer(messageFile, trailer string) error {
	content, err := os.ReadFile(messageFile)
	if err != nil {
		return err
	}
	return os.WriteFile(messageFile, []byte(strings.TrimRight(string(content), "\n")+"\n\n"+trailer+"\n"), 0644)
}

var testConfig = []byte("hooks: {}\n")

func newTestAttestation(f *fakeGit) *AttestationService {
	s := NewAttestationService(f, f, "1.3.0")
	s.now = func() time.Time { return time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC) }
	return s
}

func writeMessage(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readMessage(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

var passing = []domain.ExecutionResult{{Success: true}, {Success: true}}

func TestAttestation_RecordThenStampAddsTrailer(t *testing.T) {
	f := &fakeGit{indexTree: "tree-a"}
	s := newTestAttestation(f)

	if err := s.Record(passing, testConfig); err != nil {
		t.Fatal(err)
	}
	msg := writeMessage(t, "feat: add thing\n")
	outcome, err := s.Stamp(msg, testConfig, "")
	if err != nil || outcome != StampAdded {
		t.Fatalf("outcome %v, err %v; want StampAdded", outcome, err)
	}

	want := "Quality-Gate: v1.3.0; tree=tree-a; config=" + HashConfig(testConfig) + "; checks=2/2"
	if got := readMessage(t, msg); !strings.Contains(got, want) {
		t.Fatalf("message %q lacks trailer %q", got, want)
	}
	if f.attestation != nil {
		t.Fatal("attestation must be consumed after stamping")
	}
}

func TestAttestation_RecordRefusesFailedRun(t *testing.T) {
	f := &fakeGit{indexTree: "tree-a"}
	err := newTestAttestation(f).Record([]domain.ExecutionResult{{Success: true}, {Success: false}}, testConfig)
	if err == nil || f.attestation != nil {
		t.Fatalf("failed run was attested (err %v)", err)
	}
}

func TestAttestation_StampOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		record     bool
		indexAfter string
		config     []byte
		message    string
		skip       string
		want       StampOutcome
		wantInMsg  string
	}{
		{name: "no attestation", message: "fix: x\n", want: StampNoAttestation},
		{name: "index changed after checks", record: true, indexAfter: "tree-b", message: "fix: x\n", want: StampStale},
		{name: "config changed after checks", record: true, config: []byte("changed"), message: "fix: x\n", want: StampStale},
		{name: "empty message is left alone", record: true, message: "\n# comment only\n", want: StampEmptyMessage},
		{name: "skip reason", message: "fix: x\n", skip: "prod  hotfix", want: StampSkipTrailerAdded, wantInMsg: "Quality-Gate-Skipped: prod hotfix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeGit{indexTree: "tree-a"}
			s := newTestAttestation(f)
			if tt.record {
				if err := s.Record(passing, testConfig); err != nil {
					t.Fatal(err)
				}
			}
			if tt.indexAfter != "" {
				f.indexTree = tt.indexAfter
			}
			config := testConfig
			if tt.config != nil {
				config = tt.config
			}

			msg := writeMessage(t, tt.message)
			outcome, err := s.Stamp(msg, config, tt.skip)
			if err != nil || outcome != tt.want {
				t.Fatalf("outcome %v, err %v; want %v", outcome, err, tt.want)
			}
			got := readMessage(t, msg)
			if tt.wantInMsg != "" && !strings.Contains(got, tt.wantInMsg) {
				t.Fatalf("message %q lacks %q", got, tt.wantInMsg)
			}
			if tt.want != StampAdded && tt.want != StampSkipTrailerAdded && got != tt.message {
				t.Fatalf("message modified to %q", got)
			}
			if f.attestation != nil {
				t.Fatal("attestation must be consumed")
			}
		})
	}
}

func TestAttestation_Verify(t *testing.T) {
	trailer := func(tree string) string {
		return FormatTrailer(domain.Attestation{Version: "1.3.0", Tree: tree, ConfigSHA: HashConfig(testConfig), Passed: 1, Total: 1})
	}
	f := &fakeGit{
		order: []string{"ok", "none", "amended", "config", "skipped", "skipped-amend", "garbage"},
		commits: map[string]fakeCommit{
			"ok":            {tree: "t1", config: testConfig, message: "feat: a\n\n" + trailer("t1")},
			"none":          {tree: "t2", config: testConfig, message: "feat: b\n"},
			"amended":       {tree: "t3", config: testConfig, message: "feat: c\n\n" + trailer("old")},
			"config":        {tree: "t4", config: []byte("other"), message: "feat: d\n\n" + trailer("t4")},
			"skipped":       {tree: "t5", config: testConfig, message: "fix: e\n\nQuality-Gate-Skipped: hotfix\n"},
			"skipped-amend": {tree: "t6", config: testConfig, message: "fix: f\n\n" + trailer("old") + "\nQuality-Gate-Skipped: hotfix\n"},
			"garbage":       {tree: "t7", config: testConfig, message: "fix: g\n\nQuality-Gate: yes\n"},
		},
	}

	want := map[string]domain.CommitStatus{
		"ok":            domain.StatusAttested,
		"none":          domain.StatusMissing,
		"amended":       domain.StatusTreeMismatch,
		"config":        domain.StatusConfigMismatch,
		"skipped":       domain.StatusSkipped,
		"skipped-amend": domain.StatusSkipped,
		"garbage":       domain.StatusMalformed,
	}

	for _, allowSkip := range []bool{false, true} {
		results, err := newTestAttestation(f).Verify("range", "quality.yml", allowSkip)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range results {
			if r.Status != want[r.Commit] {
				t.Errorf("%s: status %s, want %s (%s)", r.Commit, r.Status, want[r.Commit], r.Detail)
			}
			wantAccepted := r.Status == domain.StatusAttested || (allowSkip && r.Status == domain.StatusSkipped)
			if r.Accepted != wantAccepted {
				t.Errorf("%s (allowSkip=%v): accepted %v, want %v", r.Commit, allowSkip, r.Accepted, wantAccepted)
			}
		}
	}
}

func TestParseTrailer_RoundTrip(t *testing.T) {
	a := domain.Attestation{Version: "1.3.0", Tree: "abc", ConfigSHA: "sha256:def", Passed: 3, Total: 3}
	value := strings.TrimPrefix(FormatTrailer(a), domain.AttestationTrailer+": ")
	got, err := ParseTrailer(value)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tree != a.Tree || got.ConfigSHA != a.ConfigSHA || got.Passed != 3 || got.Total != 3 || got.Version != "v1.3.0" {
		t.Fatalf("round trip produced %+v", got)
	}
}
