package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/infra/logger"
	"github.com/dmux/go-quality-gate/internal/repository"
)

// ToolManagerService is responsible for managing tools.

type ToolManagerService struct {
	shellRunner repository.ShellRunner
	logger      logger.Logger
	state       repository.ToolStateRepository
}

// NewToolManagerService creates a new ToolManagerService.

func NewToolManagerService(shellRunner repository.ShellRunner, logger logger.Logger) *ToolManagerService {
	return &ToolManagerService{shellRunner: shellRunner, logger: logger}
}

// NewCachingToolManagerService creates a ToolManagerService that skips tool
// checks when the same tools configuration was previously validated.
func NewCachingToolManagerService(shellRunner repository.ShellRunner, logger logger.Logger, state repository.ToolStateRepository) *ToolManagerService {
	return &ToolManagerService{shellRunner: shellRunner, logger: logger, state: state}
}

// EnsureToolsInstalled checks if all tools are installed. When policy is
// config.ToolsPolicyRecommend, a missing tool is only reported (with its
// install command) rather than installed automatically; any other value
// (including "") keeps the default auto-install behavior.
func (s *ToolManagerService) EnsureToolsInstalled(tools []domain.Tool, policy string) error {
	// Normalize before hashing so equivalent policies ("" and "install") can
	// never produce two different cache fingerprints for the same tools.
	if policy == "" {
		policy = config.ToolsPolicyInstall
	}

	toolsHash := hashTools(tools, policy)

	if s.state != nil {
		cachedHash, cacheErr := s.state.LoadToolsHash()
		if cacheErr == nil && cachedHash == toolsHash {
			return nil
		}
	}

	stillMissing := false

	for _, tool := range tools {
		s.logger.StartSpinner(fmt.Sprintf("Checking if %s is installed...", tool.Name))

		startTime := time.Now()
		_, err := s.shellRunner.Run(tool.CheckCommand)
		checkDuration := time.Since(startTime)

		s.logger.StopSpinner()

		if err != nil {
			if policy == config.ToolsPolicyRecommend {
				stillMissing = true
				s.logger.Print("⚠️  %s is not installed. Recommended install command: %s\n", tool.Name, tool.InstallCommand)
				continue
			}

			s.logger.StartSpinner(fmt.Sprintf("Installing %s...", tool.Name))

			installStartTime := time.Now()
			output, err := s.shellRunner.Run(tool.InstallCommand)
			installDuration := time.Since(installStartTime)

			s.logger.StopSpinner()

			if err != nil {
				return fmt.Errorf("failed to install %s: %w\n%s", tool.Name, err, output)
			}
			s.logger.Print("✅ %s installed successfully (%v)\n", tool.Name, installDuration.Round(time.Millisecond))
		} else {
			s.logger.Print("✅ %s is already installed (%v)\n", tool.Name, checkDuration.Round(time.Millisecond))
		}
	}

	// Caching a run that ended with unresolved tools would silence the
	// recommendation on every later run, since recommend mode never installs
	// anything to make the state true.
	//
	// Cache failures must not prevent the quality gate from running. They only
	// cause the tools to be checked again on the next execution.
	if s.state != nil && !stillMissing {
		_ = s.state.SaveToolsHash(toolsHash)
	}
	return nil
}

// hashTools fingerprints a tools configuration, together with the normalized
// policy it was validated under, for the install-check cache. Without the
// policy, tools accepted under "install" would satisfy the cache after a
// switch to "recommend" and their recommendations would never be printed.
//
// The marshal error is intentionally ignored: domain.Tool has only string
// fields, so json.Marshal on a []domain.Tool can never fail.
func hashTools(tools []domain.Tool, policy string) string {
	content, _ := json.Marshal(struct {
		Tools  []domain.Tool `json:"tools"`
		Policy string        `json:"policy"`
	}{Tools: tools, Policy: policy})
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}
