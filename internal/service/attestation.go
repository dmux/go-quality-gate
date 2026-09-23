package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/repository"
)

// SkipEnvVar lets a developer bypass the gates while leaving an auditable
// trailer on the commit instead of a silent --no-verify.
const SkipEnvVar = "QG_SKIP"

// StampOutcome describes what the commit-msg hook did to the message.
type StampOutcome int

const (
	StampAdded StampOutcome = iota
	StampSkipTrailerAdded
	StampNoAttestation
	StampStale
	StampEmptyMessage
)

// AttestationService watermarks commits whose content passed the quality
// gates and verifies those watermarks later, typically in CI.
type AttestationService struct {
	commits repository.CommitRepository
	state   repository.AttestationStateRepository
	version string
	now     func() time.Time
}

// NewAttestationService creates a new AttestationService.
func NewAttestationService(commits repository.CommitRepository, state repository.AttestationStateRepository, version string) *AttestationService {
	return &AttestationService{commits: commits, state: state, version: version, now: time.Now}
}

// HashConfig returns the canonical hash of a quality.yml content.
func HashConfig(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

// Record stores an attestation bound to the current index tree. It is only
// written when every result succeeded.
func (s *AttestationService) Record(results []domain.ExecutionResult, config []byte) error {
	for _, result := range results {
		if !result.Success {
			return fmt.Errorf("refusing to attest a failed run")
		}
	}

	tree, err := s.commits.WriteTree()
	if err != nil {
		return fmt.Errorf("failed to compute index tree: %w", err)
	}

	content, err := json.Marshal(domain.Attestation{
		Version:   s.version,
		Tree:      tree,
		ConfigSHA: HashConfig(config),
		Passed:    len(results),
		Total:     len(results),
		At:        s.now().UTC(),
	})
	if err != nil {
		return err
	}
	return s.state.SaveAttestation(content)
}

// Stamp adds the watermark trailer to a commit message file when a fresh
// attestation matches the content being committed. skipReason, when set,
// records a deliberate bypass instead.
func (s *AttestationService) Stamp(messageFile string, config []byte, skipReason string) (StampOutcome, error) {
	// Consume the attestation whatever happens so it can never be reused.
	defer s.state.DeleteAttestation()

	message, err := os.ReadFile(messageFile)
	if err != nil {
		return StampNoAttestation, err
	}
	// Adding a trailer to an empty message would turn an aborted commit into
	// a real one.
	if isEmptyMessage(string(message)) {
		return StampEmptyMessage, nil
	}

	if skipReason != "" {
		trailer := fmt.Sprintf("%s: %s", domain.SkipTrailer, sanitizeTrailerValue(skipReason))
		return StampSkipTrailerAdded, s.commits.AddTrailer(messageFile, trailer)
	}

	raw, err := s.state.LoadAttestation()
	if err != nil {
		return StampNoAttestation, err
	}
	if raw == nil {
		return StampNoAttestation, nil
	}

	var attestation domain.Attestation
	if err := json.Unmarshal(raw, &attestation); err != nil {
		return StampStale, nil
	}

	tree, err := s.commits.WriteTree()
	if err != nil {
		return StampStale, fmt.Errorf("failed to compute index tree: %w", err)
	}
	if attestation.Tree != tree || attestation.ConfigSHA != HashConfig(config) {
		return StampStale, nil
	}

	return StampAdded, s.commits.AddTrailer(messageFile, FormatTrailer(attestation))
}

// Verify checks the watermark of every non-merge commit in rangeSpec.
// configPath is the quality.yml path relative to the repository root.
func (s *AttestationService) Verify(rangeSpec, configPath string, allowSkip bool) ([]domain.CommitVerification, error) {
	commits, err := s.commits.RevList(rangeSpec)
	if err != nil {
		return nil, err
	}

	var verifications []domain.CommitVerification
	for _, commit := range commits {
		verification, err := s.verifyCommit(commit, configPath, allowSkip)
		if err != nil {
			return nil, err
		}
		verifications = append(verifications, verification)
	}
	return verifications, nil
}

func (s *AttestationService) verifyCommit(commit, configPath string, allowSkip bool) (domain.CommitVerification, error) {
	result := domain.CommitVerification{Commit: commit}

	message, err := s.commits.CommitMessage(commit)
	if err != nil {
		return result, err
	}
	value, hasAttestation := lastTrailer(message, domain.AttestationTrailer)
	skipReason, hasSkip := lastTrailer(message, domain.SkipTrailer)

	status, detail, err := s.checkAttestation(commit, configPath, value, hasAttestation)
	if err != nil {
		return result, err
	}

	switch {
	case status == domain.StatusAttested:
		result.Status, result.Accepted = status, true
	case hasSkip:
		// A skip trailer wins over a stale attestation left behind by an
		// amended commit.
		result.Status, result.Accepted, result.SkipReason = domain.StatusSkipped, allowSkip, skipReason
	default:
		result.Status, result.Detail = status, detail
	}
	return result, nil
}

func (s *AttestationService) checkAttestation(commit, configPath, value string, present bool) (domain.CommitStatus, string, error) {
	if !present {
		return domain.StatusMissing, "no " + domain.AttestationTrailer + " trailer", nil
	}

	attestation, err := ParseTrailer(value)
	if err != nil {
		return domain.StatusMalformed, err.Error(), nil
	}

	tree, err := s.commits.CommitTree(commit)
	if err != nil {
		return "", "", err
	}
	if attestation.Tree != tree {
		return domain.StatusTreeMismatch, fmt.Sprintf("attested tree %s, commit tree %s", short(attestation.Tree), short(tree)), nil
	}

	config, err := s.commits.ShowFile(commit, configPath)
	if err != nil {
		return "", "", err
	}
	if config == nil || attestation.ConfigSHA != HashConfig(config) {
		return domain.StatusConfigMismatch, fmt.Sprintf("attested with a %s different from the committed one", configPath), nil
	}

	return domain.StatusAttested, "", nil
}

// FormatTrailer renders the watermark trailer for an attestation.
func FormatTrailer(a domain.Attestation) string {
	return fmt.Sprintf("%s: v%s; tree=%s; config=%s; checks=%d/%d",
		domain.AttestationTrailer, strings.TrimPrefix(a.Version, "v"), a.Tree, a.ConfigSHA, a.Passed, a.Total)
}

// ParseTrailer parses the value of a Quality-Gate trailer.
func ParseTrailer(value string) (domain.Attestation, error) {
	var a domain.Attestation
	parts := strings.Split(value, ";")
	a.Version = strings.TrimSpace(parts[0])

	for _, part := range parts[1:] {
		key, val, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "tree":
			a.Tree = val
		case "config":
			a.ConfigSHA = val
		case "checks":
			passed, total, _ := strings.Cut(val, "/")
			a.Passed, _ = strconv.Atoi(passed)
			a.Total, _ = strconv.Atoi(total)
		}
	}

	if a.Tree == "" || a.ConfigSHA == "" {
		return a, fmt.Errorf("trailer %q lacks tree or config", value)
	}
	return a, nil
}

// lastTrailer returns the value of the last line of message that starts
// with "key:".
func lastTrailer(message, key string) (string, bool) {
	lines := strings.Split(message, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		name, value, ok := strings.Cut(lines[i], ":")
		if ok && strings.EqualFold(strings.TrimSpace(name), key) {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}

func isEmptyMessage(message string) bool {
	for _, line := range strings.Split(message, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && strings.Contains(trimmed, ">8") {
			break
		}
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return false
		}
	}
	return true
}

func sanitizeTrailerValue(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func short(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}
