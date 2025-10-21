package service

import (
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
}

// NewToolManagerService creates a new ToolManagerService.

func NewToolManagerService(shellRunner repository.ShellRunner, logger logger.Logger) *ToolManagerService {
	return &ToolManagerService{shellRunner: shellRunner, logger: logger}
}

// EnsureToolsInstalled checks if all tools are installed and installs them if they are not.

func (s *ToolManagerService) EnsureToolsInstalled(tools []domain.Tool) error {
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
	return nil
}
