package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

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

// EnsureToolsInstalled checks if all tools are installed and installs them if they are not.

func (s *ToolManagerService) EnsureToolsInstalled(tools []domain.Tool) error {
	toolsHash, err := hashTools(tools)
	if err != nil {
		return fmt.Errorf("failed to hash tools configuration: %w", err)
	}

	if s.state != nil {
		cachedHash, cacheErr := s.state.LoadToolsHash()
		if cacheErr == nil && cachedHash == toolsHash {
			return nil
		}
	}

	for _, tool := range tools {
		s.logger.StartSpinner(fmt.Sprintf("Checking if %s is installed...", tool.Name))

		startTime := time.Now()
		_, err := s.shellRunner.Run(tool.CheckCommand)
		checkDuration := time.Since(startTime)

		s.logger.StopSpinner()

		if err != nil {
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

	// Cache failures must not prevent the quality gate from running. They only
	// cause the tools to be checked again on the next execution.
	if s.state != nil {
		_ = s.state.SaveToolsHash(toolsHash)
	}
	return nil
}

func hashTools(tools []domain.Tool) (string, error) {
	content, err := json.Marshal(tools)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:]), nil
}
