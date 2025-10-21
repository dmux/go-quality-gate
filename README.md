# Go Quality Gate

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

````yaml
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
```Como Usar1. Pré-requisitosGo 1.18+ (apenas para compilar a ferramenta).Gerenciadores de pacotes para as linguagens do seu projeto (ex: pip para Python, npm para Node.js).2. InstalaçãoA instalação é feita em dois passos: compilar o programa e depois usar o próprio programa para instalar os Git hooks.Passo 1: Compilar o executávelgo build -o quality-gate .

Isso criará um executável chamado quality-gate no diretório atual.Passo 2: Instalar os Git Hooks./quality-gate --install

O programa irá configurar automaticamente os hooks pre-commit e pre-push.3. Comandos Avançados./quality-gate --init: (Experimental) Tenta analisar a estrutura do seu projeto e gera um arquivo quality.yml inicial com sugestões../quality-gate --fix: Executa os comandos de correção automática (fix_command) definidos no seu quality.yml../quality-gate pre-commit --output=json: Executa o hook especificado e retorna o resultado em formato JSON.4. Configuração (quality.yml)A configuração agora é dividida em duas seções principais: tools (para o gerenciamento de dependências) e hooks (para as verificações).tools: Uma lista de ferramentas necessárias para o projeto.name: Nome legível da ferramenta.check_command: Um comando que retorna sucesso (código de saída 0) se a ferramenta estiver instalada (ex: gitleaks version).install_command: O comando a ser executado para instalar a ferramenta se o check_command falhar.hooks: A configuração das verificações, como antes.Exemplo Abrangente:tools:

- name: "Gitleaks"
  check_command: "gitleaks version"
  install_command: "go install [github.com/zricethezav/gitleaks/v8@latest](https://github.com/zricethezav/gitleaks/v8@latest)"
- name: "Ruff (Python Linter/Formatter)"
  check_command: "ruff --version"
  install_command: "pip install ruff"
- name: "Prettier (Code Formatter)"
  check_command: "npx prettier --version"
  install_command: "npm install --global prettier"

hooks:
security:
pre-commit: - name: "🔒 Verificação de Segredos (Gitleaks)"
command: "gitleaks detect --no-git --source . --verbose"
output_rules:
on_failure_message: "Vazamento de segredo detectado! Revise o código antes de comitar."

python-backend:
pre-commit: - name: "🎨 Verificação de Formato (Ruff)"
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
pre-commit: - name: "🎨 Formatação (Prettier)"
command: "npx prettier --check 'frontend/**/\*.{ts,tsx}'"
fix_command: "npx prettier --write 'frontend/**/\*.{ts,tsx}'"

## 📋 Comandos Disponíveis

| Comando | Descrição | Exemplo |
|---------|-----------|---------|
| `--install` | Instala Git hooks no repositório | `./quality-gate --install` |
| `--init` | Gera quality.yml inicial | `./quality-gate --init` |
| `--fix` | Executa correções automáticas | `./quality-gate --fix pre-commit` |
| `--output=json` | Output estruturado para CI/CD | `./quality-gate --output=json pre-commit` |

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
````

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
