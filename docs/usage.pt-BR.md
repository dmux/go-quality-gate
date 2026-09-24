[![en](https://img.shields.io/badge/lang-en-red.svg)](usage.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](usage.pt-BR.md)

# Go Quality Gate — Guia de Uso

Um guia curto e prático para usar o quality-gate no dia a dia. Para todas as opções, veja o [README](../README.pt-BR.md).

## 1. Instale o binário

```bash
# macOS (Apple Silicon) — outras plataformas: veja a página de releases
curl -fsSL -o quality-gate https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-darwin-arm64
chmod +x quality-gate && sudo mv quality-gate /usr/local/bin/

# ou, com Go
go install github.com/dmux/go-quality-gate/cmd/quality-gate@latest

quality-gate --version
```

## 2. Crie a configuração

Na raiz do repositório:

```bash
quality-gate --init
```

Ele detecta as linguagens do projeto e gera um `quality.yml`. Revise e faça commit dele.

## 3. Instale os hooks

```bash
quality-gate --install
```

São instalados três hooks: `pre-commit` (roda os checks), `commit-msg` (adiciona o watermark) e `pre-push`.

Em projetos Python, o `--init` também adiciona a auditoria de dependências `pip-audit` **tanto no `pre-commit` quanto no `pre-push`**, para que bibliotecas vulneráveis sejam pegas quando você commita e de novo logo antes de saírem da sua máquina.

> Já usava o quality-gate antes da v1.3.0? Rode `--install` de novo para ganhar o hook `commit-msg`.

Para aplicar o gate em **todos** os repositórios da máquina que tenham `quality.yml`, rode:

```bash
quality-gate --install --global
```

## 4. Confira a instalação

```bash
quality-gate doctor
```

```
✅ binary on PATH: /usr/local/bin/quality-gate
✅ pre-commit hook: .git/hooks/pre-commit
✅ commit-msg hook: .git/hooks/commit-msg
✅ pre-push hook: .git/hooks/pre-push
✅ quality.yml
```

## 5. Faça commit normalmente

```bash
git add .
git commit -m "feat: adiciona login"
```

O que acontece por trás:

1. **`pre-commit`** roda os checks do `quality.yml`.
   - Um check falhando **bloqueia** o commit. Corrija (ou rode `quality-gate --fix pre-commit`) e tente de novo.
   - Se todos passarem, o quality-gate guarda um hash do conteúdo em stage e do `quality.yml`.
2. **Você escreve a mensagem** (`-m` ou editor).
3. **`commit-msg`** confirma que nada mudou desde os checks e acrescenta o watermark:
   ```
   feat: adiciona login

   Quality-Gate: v1.3.0; tree=db228f6d…; config=sha256:4feac6c2…; checks=3/3
   ```
   Aparece a mensagem `🔏 Commit watermarked by quality-gate.`
4. **O commit é criado** com o watermark.

Para conferir:

```bash
git log -1 --format='%(trailers)'
quality-gate verify --range HEAD
```

## 6. Situações especiais

| Situação | O que acontece |
|---|---|
| `git commit --no-verify` | Nenhum hook roda; o commit sai **sem** watermark e o CI rejeita (`missing`). |
| `QG_SKIP="hotfix produção" git commit -m …` | Os checks não rodam, mas o commit ganha `Quality-Gate-Skipped: hotfix produção`. O CI aceita só com `--policy allow-skip`. |
| `git commit --amend` | Os hooks rodam de novo e o watermark é substituído. Com `--no-verify`, o watermark antigo deixa de bater (`tree-mismatch`). |
| Rebase / cherry-pick que muda o conteúdo | O watermark antigo deixa de bater (`tree-mismatch`); refaça o commit com os hooks ativos. |
| Arquivos em stage mudaram entre os checks e a mensagem | Sem watermark, com um aviso. Faça o commit de novo. |
| Mensagem vazia (editor fechado) | O Git aborta normalmente; nada é adicionado. |
| Commit feito por agente de IA via MCP | O `run_quality_checks` também grava o watermark. |

O hook `commit-msg` nunca bloqueia o commit; ele só avisa. Quem barra de fato é o CI.

## 7. Correções e saída JSON

```bash
quality-gate --fix pre-commit            # roda os comandos de correção (ruff format, pytest…)
quality-gate --fix pre-push              # roda pip-audit --fix: atualiza pins vulneráveis
quality-gate pre-commit                  # roda os checks manualmente
quality-gate pre-push                    # re-roda a auditoria de dependências sob demanda
quality-gate --output=json pre-commit    # saída para máquinas
```

O `--output=json` embute a saída bruta de cada comando como string em
`results[].output` (passthrough — no caso da auditoria Python é a tabela
`--aliases` do pip-audit). Para dados de vulnerabilidade parseáveis por
máquina, rode o pip-audit direto:

```bash
pip-audit -f json
```

## 8. Obrigue o uso no CI

Adicione a Action ao repositório (`.github/workflows/quality-gate.yml`):

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
          run-checks: "true"    # roda os checks de novo no CI
```

Depois, em **Settings → Branches (ou Rules)**, marque o job `quality-gate` como **required status check** da `main`. Pull requests com commits que pularam o gate não podem mais ser mergeados.

Fora do GitHub, rode o mesmo comando em qualquer CI:

```bash
quality-gate verify --range origin/main..HEAD --policy strict --output json
```

| Status | Significado |
|---|---|
| `attested` | Passou nos gates com exatamente este conteúdo ✅ |
| `skipped` | Pulado com `QG_SKIP` (aceito só com `allow-skip`) |
| `missing` | Sem watermark (hooks não instalados ou `--no-verify`) |
| `tree-mismatch` | O conteúdo mudou depois que os gates rodaram |
| `config-mismatch` | Watermark gerado com outro `quality.yml` |
| `malformed` | Watermark ilegível |

## 9. Perguntas frequentes

**O que é um trailer?** Uma linha `Chave: valor` no fim da mensagem do commit, depois de uma linha em branco — a mesma convenção do `Signed-off-by` e do `Co-authored-by`. Ele viaja junto com o commit e o Git sabe lê-lo nativamente (`git interpret-trailers`, `%(trailers)`).

**Dá para forjar o watermark?** Sim, alguém poderia digitá-lo à mão. Ele barra bypass casual e deixa rastro para auditoria; o check obrigatório no CI, que roda os gates de novo, é o que torna o gate mandatório.

**E o squash merge?** Ele cria um commit novo sem watermark. Verifique os commits do pull request, não o commit de merge — a Action já faz isso por padrão.

**A auditoria de dependências precisa de rede?** Sim. O `pip-audit` consulta PyPI/OSV, então um commit offline falha no check de audit. Pule o gate naquele commit com `QG_SKIP="offline" git commit …`, ou remova os comandos pip-audit do seu `quality.yml` se preferir auditar só no CI.
