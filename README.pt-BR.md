[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](README.pt-BR.md)

<div align="center">

<img src="docs/gopher.png" alt="Go Quality Gate Logo" width="400">

# Go Quality Gate

[![Go Version](https://img.shields.io/badge/Go-1.24.5+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen)](https://github.com/dmux/go-quality-gate)
[![Code Quality](https://img.shields.io/badge/Quality-A-brightgreen)](https://github.com/dmux/go-quality-gate)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/dmux/go-quality-gate/pulls)

</div>

**Ferramenta agnóstica de controle de qualidade com Git hooks**

Uma ferramenta de controle de qualidade de código construída em Go, distribuída como um único binário sem dependências externas de runtime. Fornece feedback visual aprimorado com spinners, timing de execução e output JSON estruturado.

## ✨ Características Principais

- **🏗️ Binário Único**: Zero dependências de runtime (Python, Node.js)
- **🔧 Setup Automático**: Instala ferramentas de qualidade automaticamente
- **🌍 Multi-linguagem**: Suporta múltiplas linguagens em um mesmo repositório
- **📊 Observabilidade**: Spinners, timing e feedback visual em tempo real
- **🔒 Segurança Integrada**: Verificação de segredos no fluxo de commit
- **⚡ Performance Nativa**: Execução instantânea sem interpretadores
- **🚀 CI/CD Ready**: Output JSON limpo para pipelines de automação

## 🚀 Quick Start

### 1. Instalação

```bash
# Clone e compile
git clone <repo>
cd go-quality-gate
go build -o quality-gate ./cmd/quality-gate

# Instale os hooks
./quality-gate --install
```

### 2. Configuração

Crie um `quality.yml` no seu projeto:

```bash
# Gera configuração inicial baseada no seu projeto
./quality-gate --init
```

### 3. Uso

```bash
# Execução automática via Git hooks
git commit -m "feat: nova funcionalidade"

# Execução manual
./quality-gate pre-commit

# Output JSON para CI/CD
./quality-gate --output=json pre-commit

# Correção automática
./quality-gate --fix pre-commit
```

## ⚙️ Configuração (quality.yml)

```yaml
tools:
  - name: "Gitleaks"
    check_command: "gitleaks version"
    install_command: "go install github.com/zricethezav/gitleaks/v8@latest"
  - name: "Ruff (Python)"
    check_command: "ruff --version"
    install_command: "pip install ruff"

hooks:
  security:
    pre-commit:
      - name: "🔒 Verificação de Segredos"
        command: "gitleaks detect --no-git --source . --verbose"
        output_rules:
          on_failure_message: "Vazamento de segredo detectado!"

  python-backend:
    pre-commit:
      - name: "🎨 Formatação (Ruff)"
        command: "ruff format ./backend --check"
        fix_command: "ruff format ./backend"
        output_rules:
          show_on: failure
          on_failure_message: "Execute './quality-gate --fix' para corrigir."

  typescript-frontend:
    pre-commit:
      - name: "🎨 Formatação (Prettier)"
        command: "npx prettier --check 'frontend/**/*.{ts,tsx}'"
        fix_command: "npx prettier --write 'frontend/**/*.{ts,tsx}'"
```

## 📘 Como Usar

### 1. Compilação

```bash
go build -o quality-gate ./cmd/quality-gate
```

Isso criará um executável chamado `quality-gate` no diretório atual.

### 2. Instalar os Git Hooks

```bash
./quality-gate --install
```

O programa irá configurar automaticamente os hooks `pre-commit` e `pre-push`.

### 3. Comandos Avançados

- **`./quality-gate --init`**: (Experimental) Tenta analisar a estrutura do seu projeto e gera um arquivo `quality.yml` inicial com sugestões.
- **`./quality-gate --fix`**: Executa os comandos de correção automática (`fix_command`) definidos no seu `quality.yml`.
- **`./quality-gate pre-commit --output=json`**: Executa o hook especificado e retorna o resultado em formato JSON.

### 4. Configuração (quality.yml)

A configuração é dividida em duas seções principais:

- **`tools`**: Lista de ferramentas necessárias para o projeto

  - `name`: Nome legível da ferramenta
  - `check_command`: Comando que retorna sucesso (código de saída 0) se a ferramenta estiver instalada
  - `install_command`: Comando executado para instalar a ferramenta se o `check_command` falhar

- **`hooks`**: Configuração das verificações de qualidade

#### Cache de Validação das Ferramentas

Na primeira execução de um `pre-commit` ou `pre-push`, o Quality Gate executa
o `check_command` de cada ferramenta configurada e instala as ferramentas
ausentes por meio do respectivo `install_command`. Depois que todas forem
validadas com sucesso, um hash SHA-256 da configuração `tools` é salvo em:

```text
.git/quality-gate/tools.sha256
```

As execuções seguintes pulam as verificações de instalação enquanto a
configuração permanecer inalterada. Alterar o `name`, `check_command` ou
`install_command` de uma ferramenta, assim como adicionar, remover ou reordenar
ferramentas, invalida o cache e faz com que todas sejam validadas novamente.

O cache só é atualizado depois de uma validação bem-sucedida. Se estiver
ausente, desatualizado, ilegível ou não puder ser gravado, o Quality Gate volta
a verificar as ferramentas normalmente, sem impedir a execução dos hooks
configurados. Para forçar uma nova validação — por exemplo, após desinstalar
manualmente uma ferramenta — remova o arquivo de cache antes de executar um
hook:

```bash
rm .git/quality-gate/tools.sha256
```

#### Exemplo Completo

```yaml
tools:
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
```

## 📋 Comandos Disponíveis

| Comando         | Descrição                                        | Exemplo                                   |
| --------------- | ------------------------------------------------ | ----------------------------------------- |
| `--install`     | Instala Git hooks no repositório                 | `./quality-gate --install`                |
| `--init`        | Gera quality.yml inicial com análise inteligente | `./quality-gate --init`                   |
| `--fix`         | Executa correções automáticas                    | `./quality-gate --fix pre-commit`         |
| `--version, -v` | Mostra informações de versão                     | `./quality-gate --version`                |
| `--output=json` | Output estruturado para CI/CD                    | `./quality-gate --output=json pre-commit` |

### 📊 Informações de Versão

```bash
# Versão simples
./quality-gate --version
# Output: quality-gate version 1.2.0

# Versão em JSON com detalhes de build
./quality-gate --version --output json
# Output:
{
  "version": "1.2.0",
  "build_date": "2025-10-21T16:34:44Z",
  "git_commit": "f7b01a2"
}
```

## 🎯 Output JSON para CI/CD

```json
{
  "status": "success",
  "results": [
    {
      "hook": {
        "Name": "🔒 Security Check",
        "Command": "gitleaks detect --source ."
      },
      "success": true,
      "output": "",
      "duration_ms": 150,
      "duration": "150ms"
    }
  ]
}
```

## 🛠️ Desenvolvimento

### Pré-requisitos

- Go 1.18+
- Git
- Gerenciadores de pacotes das linguagens do seu projeto (pip, npm, etc.)

### Setup Local

```bash
# Clone o repositório
git clone <repo>
cd go-quality-gate

# Instale dependências
go mod tidy

# Compile
go build -o quality-gate ./cmd/quality-gate

# Execute testes
go test ./...

# Teste localmente
./quality-gate --init
./quality-gate --install
```

### Arquitetura

```text
cmd/quality-gate/     # Aplicação principal
internal/
  domain/            # Entidades e regras de negócio
  service/           # Lógica de aplicação
  infra/             # Infraestrutura (git, shell, logger)
  repository/        # Interfaces de persistência
  config/            # Configuração e parsing
```

## 🤝 Contribuindo

1. **Fork** o projeto
2. **Crie** uma branch: `git checkout -b feature/nova-funcionalidade`
3. **Commit** suas mudanças: `git commit -m 'feat: nova funcionalidade'`
4. **Push** para a branch: `git push origin feature/nova-funcionalidade`
5. **Abra** um Pull Request

Veja [TODO.md](TODO.md) para roadmap detalhado e tarefas disponíveis.

## 📄 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.

---

**Versão**: v1.1.x  
**Status**: Ativo em desenvolvimento  
**Documentação completa**: [TODO.md](TODO.md)
