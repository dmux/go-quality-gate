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
- **🔒 Segurança Integrada**: Verificação de segredos + auditoria de vulnerabilidades em dependências Python (pip-audit) no fluxo de commit/push
- **⚡ Performance Nativa**: Execução instantânea sem interpretadores
- **🚀 CI/CD Ready**: Output JSON limpo para pipelines de automação
- **🤖 Servidor MCP**: Suporte ao Model Context Protocol para Agentes de IA (Cursor, Claude, etc.)

## 🚀 Quick Start

> 📖 Primeira vez? O [Guia de Uso](docs/usage.pt-BR.md) mostra a instalação e o fluxo de commit do dia a dia em poucos minutos.

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

# Rodar como Servidor Model Context Protocol (MCP) para IAs
./quality-gate mcp
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
  - name: "Pip-Audit (Python Dependency Audit)"
    check_command: "pip-audit --version"
    install_command: "pip install pip-audit"

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
      - name: "🛡️ Auditoria de Dependências (pip-audit)"
        command: "pip-audit -r requirements.txt --aliases"
        fix_command: "pip-audit --fix -r requirements.txt"
        output_rules:
          show_on: failure
          on_failure_message: "Dependências vulneráveis encontradas! Execute './quality-gate --fix' ou atualize manualmente."
    pre-push:
      - name: "🛡️ Auditoria de Dependências (pip-audit)"
        command: "pip-audit -r requirements.txt --aliases"
        fix_command: "pip-audit --fix -r requirements.txt"
        output_rules:
          show_on: failure
          on_failure_message: "Dependências vulneráveis encontradas! Execute './quality-gate --fix' ou atualize manualmente."

  typescript-frontend:
    pre-commit:
      - name: "🎨 Formatação (Prettier)"
        command: "npx prettier --check 'frontend/**/*.{ts,tsx}'"
        fix_command: "npx prettier --write 'frontend/**/*.{ts,tsx}'"
```

> A auditoria de dependências com `pip-audit` roda tanto no `pre-commit`
> quanto no `pre-push` e precisa de acesso à rede (PyPI/OSV). Vai committar
> offline? Pule o gate com `QG_SKIP="offline" git commit …` ou remova os
> comandos de audit do seu `quality.yml`.

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
  - name: "Pip-Audit (Python Dependency Audit)"
    check_command: "pip-audit --version"
    install_command: "pip install pip-audit"
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
      - name: "🛡️ Auditoria de Dependências (pip-audit)"
        command: "pip-audit -r requirements.txt --aliases"
        fix_command: "pip-audit --fix -r requirements.txt"
        output_rules:
          show_on: failure
          on_failure_message: "Dependências vulneráveis encontradas! Execute './quality-gate --fix' ou atualize manualmente."
    pre-push:
      - name: "🛡️ Auditoria de Dependências (pip-audit)"
        command: "pip-audit -r requirements.txt --aliases"
        fix_command: "pip-audit --fix -r requirements.txt"
        output_rules:
          show_on: failure
          on_failure_message: "Dependências vulneráveis encontradas! Execute './quality-gate --fix' ou atualize manualmente."

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
| `--install --global` | Aplica o gate em todos os repositórios do usuário (`core.hooksPath` global) | `./quality-gate --install --global` |
| `verify`        | Verifica o watermark dos commits (para CI)       | `./quality-gate verify --range origin/main..HEAD` |
| `doctor`        | Confere hooks, binário e quality.yml             | `./quality-gate doctor`                   |
| `mcp`           | Roda como um servidor MCP para integração com IA | `./quality-gate mcp`                      |
| `--version, -v` | Mostra informações de versão                     | `./quality-gate --version`                |
| `--output=json` | Output estruturado para CI/CD                    | `./quality-gate --output=json pre-commit` |

### 📊 Informações de Versão

```bash
# Versão simples
./quality-gate --version
# Output: quality-gate version 1.3.0

# Versão em JSON com detalhes de build
./quality-gate --version --output json
# Output:
{
  "version": "1.3.0",
  "build_date": "2025-10-21T16:34:44Z",
  "git_commit": "f7b01a2"
}
```

## 🔏 Enforcement e Watermark nos Commits

Hooks no cliente sempre podem ser contornados (`git commit --no-verify`, apagar o hook). Por isso o quality-gate trabalha em duas camadas:

**1. Watermark no cliente.** Quando todos os checks do `pre-commit` passam, o quality-gate grava uma attestation vinculada ao conteúdo exato em stage (`git write-tree`). O hook `commit-msg` então adiciona um trailer:

```
feat: adiciona login

Quality-Gate: v1.3.0; tree=db228f6d…; config=sha256:4feac6c2…; checks=3/3
```

- `--no-verify` pula os dois hooks, então o commit sai **sem** trailer.
- Amend ou rebase sem rodar os checks de novo muda o tree, e o trailer **deixa de bater**.
- Precisa pular de propósito? `QG_SKIP="hotfix produção" git commit …` pula os checks mas grava `Quality-Gate-Skipped: hotfix produção` para auditoria.

**2. Enforcement no servidor.** `quality-gate verify` confere todos os commits de um range e falha em watermarks ausentes, desatualizados ou malformados:

```bash
quality-gate verify --range origin/main..HEAD                    # strict
quality-gate verify --range origin/main..HEAD --policy allow-skip # aceita commits com QG_SKIP
quality-gate verify --range origin/main..HEAD --output json
```

Use a GitHub Action incluída no repositório e marque o job como **required status check** na branch protection / ruleset da `main`:

```yaml
name: Quality Gate
on: pull_request
jobs:
  quality-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: dmux/go-quality-gate@main
        with:
          policy: strict        # ou allow-skip
          run-checks: "true"    # re-executa os checks, então um trailer escrito à mão não passa
```

> O watermark do cliente barra bypass casual; não é prova criptográfica, já que qualquer um pode digitar um trailer. O check obrigatório no CI, que também re-executa os gates, é o que torna o gate mandatório. Squash merge cria um commit novo sem trailer, então verifique os commits do pull request, não o commit de merge.

**Commit passo a passo**

```bash
quality-gate --install   # uma vez por repositório (instala pre-commit, commit-msg, pre-push)
quality-gate doctor      # confere binário, hooks e quality.yml

git add .
git commit -m "feat: adiciona login"
```

1. `pre-commit` roda os checks. Uma falha bloqueia o commit; se passar, os hashes do tree em stage e do `quality.yml` são guardados.
2. Você escreve a mensagem.
3. `commit-msg` confirma que nada mudou e acrescenta o trailer `Quality-Gate:` (`🔏 Commit watermarked by quality-gate.`).
4. O commit é criado. Confira com `git log -1 --format='%(trailers)'` ou `quality-gate verify --range HEAD`.

| Situação | Resultado |
|---|---|
| `git commit --no-verify` | Sem trailer → `missing` no CI |
| `QG_SKIP="motivo" git commit` | `Quality-Gate-Skipped: motivo` → aceito só com `--policy allow-skip` |
| `--amend` / rebase que muda o conteúdo sem os hooks | O trailer antigo deixa de bater → `tree-mismatch` |
| Conteúdo em stage mudou entre os checks e a mensagem | Sem trailer, com aviso — faça o commit de novo |
| Commit pelo servidor MCP | Também recebe o watermark |

O hook `commit-msg` nunca bloqueia o commit; quem barra é o CI.

**Tornando a instalação automática**

- `quality-gate --install --global` configura um `core.hooksPath` global cujos hooks só agem em repositórios com `quality.yml` (e continuam rodando os hooks locais do repositório). Pode ser distribuído pela TI/MDM para todas as máquinas.
- Adicione `quality-gate --install` ao bootstrap do projeto (script `"prepare"` no `package.json`, `make setup`, etc.).
- `quality-gate doctor` aponta hooks ausentes ou adulterados.
- O servidor MCP também grava a attestation, então commits feitos por agentes de IA saem com watermark.

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

## 🤖 Integração com Model Context Protocol (MCP)

O Go Quality Gate pode atuar como um servidor MCP, fornecendo aos assistentes de IA (como Cursor, Claude Desktop ou Cline) a capacidade nativa de interagir com as ferramentas de qualidade do seu repositório.

### Conectando a um Agente
Para usar o `go-quality-gate` com ferramentas como o Cursor, basta configurar o servidor MCP via standard I/O (stdio). No seu projeto (ou na interface do Cursor), adicione ou crie um `.mcp.json` na raiz:

```json
{
  "mcpServers": {
    "go-quality-gate": {
      "command": "./quality-gate",
      "args": ["mcp"]
    }
  }
}
```

O servidor expõe as seguintes "Tools" (ferramentas) para a sua IA:
1. `run_quality_checks`: Executa linters e formatações, retornando diagnósticos parseados para a IA corrigir.
2. `run_auto_fix`: Permite que a IA acione os comandos de formatação automáticos configurados no `quality.yml`.

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
