package domain

import "time"

const (
	// AttestationTrailer marks a commit whose exact content passed the gates.
	AttestationTrailer = "Quality-Gate"
	// SkipTrailer marks a commit whose gates were deliberately skipped.
	SkipTrailer = "Quality-Gate-Skipped"
)

// Attestation records that the quality gates passed for a given index tree
// and configuration.
type Attestation struct {
	Version   string    `json:"version"`
	Tree      string    `json:"tree"`
	ConfigSHA string    `json:"config_sha"`
	Passed    int       `json:"passed"`
	Total     int       `json:"total"`
	At        time.Time `json:"at"`
}

// CommitStatus is the verification outcome of a single commit.
type CommitStatus string

const (
	StatusAttested       CommitStatus = "attested"
	StatusSkipped        CommitStatus = "skipped"
	StatusMissing        CommitStatus = "missing"
	StatusMalformed      CommitStatus = "malformed"
	StatusTreeMismatch   CommitStatus = "tree-mismatch"
	StatusConfigMismatch CommitStatus = "config-mismatch"
)

// CommitVerification is the result of verifying one commit's watermark.
type CommitVerification struct {
	Commit     string       `json:"commit"`
	Status     CommitStatus `json:"status"`
	Accepted   bool         `json:"accepted"`
	Detail     string       `json:"detail,omitempty"`
	SkipReason string       `json:"skip_reason,omitempty"`
}
