package service

import (
	"os"
)

const (
	qualityYmlContent = `tools:
  - name: "Gitleaks"
    check_command: "gitleaks version"
    install_command: "go install github.com/zricethezav/gitleaks/v8@latest"
  - name: "Ruff (Python Linter/Formatter)"
    check_command: "ruff --version"
    install_command: "pip install ruff"
  - name: "Prettier (Code Formatter)"
    check_command: "npx prettier --version"
    install_command: "npm install --global prettier"

hooks:
  security:
    pre-commit:
      - name: "🔒 Verificação de Segredos (Gitleaks)"
        command: "gitleaks detect --no-git --source . --verbose"
        output_rules:
          on_failure_message: "Vazamento de segredo detectado! Revise o código antes de comitar."

  python-backend:
    pre-commit:
      - name: "🎨 Verificação de Formato (Ruff)"
        command: "ruff format ./backend --check"
        fix_command: "ruff format ./backend"
        output_rules:
          show_on: failure 
          on_failure_message: "Código fora do padrão. Execute './quality-gate --fix' para corrigir."

      - name: "🧪 Testes (Pytest)"
        command: "pytest ./backend"
        output_rules:
          show_on: always 

  typescript-frontend:
    pre-commit:
      - name: "🎨 Formatação (Prettier)"
        command: "npx prettier --check 'frontend/**/*.{ts,tsx}'"
        fix_command: "npx prettier --write 'frontend/**/*.{ts,tsx}'"
`
)

// InitService is responsible for initializing the quality.yml file.

type InitService struct{}

// NewInitService creates a new InitService.

func NewInitService() *InitService {
	return &InitService{}
}

// Init creates the quality.yml file.

func (s *InitService) Init() error {
	return os.WriteFile("quality.yml", []byte(qualityYmlContent), 0644)
}
