# Boas práticas Go — pokemon-self-play

Guia de referência para desenvolvimento e revisão de código neste projeto.

---

## 1. Estrutura de pacotes

Siga a convenção `internal/` para código não exportado fora do módulo.
Cada pacote tem uma responsabilidade: `cards` só conhece cartas, `game` só conhece o motor de regras, `bot` consome `game` mas não o contrário.

```
Dependências permitidas:
  bot  → game, cards, deck
  game → cards
  deck → cards
  cmd  → tudo
```

Nunca importe um pacote filho de volta no pai (ciclo). Use interfaces para inverter dependências quando necessário.

---

## 2. Erros

**Sempre envolva erros com contexto:**

```go
// ruim
return err

// bom
return fmt.Errorf("fetch card %s: %w", id, err)
```

**`%w` (não `%v`) para erros que o chamador possa inspecionar com `errors.Is`/`errors.As`.**

Erros de validação de input do usuário: mensagem legível em PT, sem stack trace.
Erros internos inesperados: wrappe até o ponto de entrada (`cmd/`) e use `log.Fatal`.

**Não ignore erros silenciosamente:**

```go
// proibido
json.NewDecoder(r.Body).Decode(&req)

// obrigatório
if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
```

A exceção explícita deste projeto: `_ = g.EndTurn(b.Player)` no piloto do bot, quando a falha é impossível pelo fluxo — documente o porquê.

---

## 3. Nomes

| O quê | Convenção | Exemplo |
|---|---|---|
| Exportado | PascalCase | `PlaceActive`, `PokemonInPlay` |
| Não exportado | camelCase | `requireTurn`, `handCard` |
| Constantes de tipo | PascalCase agrupado em `const (…)` | `PhaseSetup`, `CondAsleep` |
| Variáveis de loop | `i`, `p`, `id` — curtos e óbvios | — |
| Receptores | 1-2 letras, consistente por tipo | `g *Game`, `b *Pilot` |

Evite abreviações inventadas; prefira nomes que se leem como prosa:
`isBasicPokemon(c)`, não `chkBasPkm(c)`.

---

## 4. Funções e métodos

- **Uma função, uma responsabilidade.** Se precisar de um comentário para dividir seções com `// ----`, provavelmente são duas funções.
- **Retorne cedo.** Guard clauses no topo, lógica principal no final.

```go
// bom
func (g *Game) PlaceActive(p, handIdx int) error {
    if g.Phase != PhaseSetup {
        return fmt.Errorf("fora da fase de setup")
    }
    if ps.Active != nil {
        return fmt.Errorf("jogador %d já tem Ativo", p+1)
    }
    // lógica principal aqui
}
```

- **Limite de argumentos:** mais de 3-4 parâmetros relacionados → struct.
- **Funções puras quando possível:** `BaseDamage`, `CostPaid`, `isBasicPokemon` são fáceis de testar exatamente por não dependerem de estado externo.

---

## 5. Concorrência

O servidor HTTP (`cmd/play`) usa um único `sync.Mutex` cobrindo todo o estado da partida. Regra:

```go
// TODA modificação de g (Game) passa por s.mu.Lock()
s.mu.Lock()
defer s.mu.Unlock()
```

**Não leia `s.g` fora do lock.** Se precisar expor dados para outro goroutine, copie os valores necessários dentro do lock.

Não adicione goroutines ao motor de regras (`internal/game`). O motor é single-threaded por design — determinismo por seed depende disso.

---

## 6. Testes

- Prefira testes de integração sobre unit tests para o motor: `game_test.go` joga partidas completas, não testa métodos isolados.
- Use `rand.New(rand.NewSource(seed))` fixo nos testes — nunca `rand.Intn` global.
- Nomeie subtestes com o cenário, não com o método:

```go
// ruim
t.Run("TestAttack", func(t *testing.T) { ... })

// bom
t.Run("ataque com Fraqueza dobra dano", func(t *testing.T) { ... })
```

- Rode `task check` antes de commitar: `go build + go vet + go test`.

---

## 7. Comentários

Comente o **porquê**, não o **o quê**. Identificadores bem nomeados já explicam o quê.

```go
// ruim: "remove a carta da mão"
g.removeFromHand(p, handIdx)

// bom: explica invariante não óbvia
// TCGdex usa "Normal" para Energia Básica, não "Basic"
func isBasicEnergy(c *cards.Card) bool {
    return c.Category == CategoryEnergy && c.EnergyType != "Special"
}
```

Pacotes exportados usam godoc (`// NomeDaFunção …`); código interno não precisa de comentário em cada função.

---

## 8. Dependências

Política: **zero dependências externas no módulo principal** (só stdlib).

Exceção justificada: `go-task` é ferramenta de dev, não entra no `go.mod` do projeto.

Antes de adicionar qualquer `go get`:
1. A stdlib resolve? (`encoding/json`, `net/http`, `sync`, `math/rand`, `embed`…)
2. São menos de ~50 linhas para reimplementar?
3. A dependência tem licença compatível e manutenção ativa?

---

## 9. Performance

Não otimize antes de medir. Para este projeto, o gargalo quase sempre será I/O (rede, disco) — não CPU.

Padrões já usados que evitam alocações desnecessárias:
- `append([]string{}, slice...)` para copiar slices sem compartilhar backing array.
- `sort.Strings` + `sort.Slice` em vez de maps para pequenas coleções.
- JSON encode/decode direto no `http.ResponseWriter` sem buffer intermediário.

Se `go tool pprof` apontar um hot path real, adicione um comentário `// ponytail: <simplificação atual, upgrade quando X>` antes de otimizar.

---

## 10. Checklist de PR

- [ ] `task check` passa (build + vet + tests)
- [ ] Nenhum `fmt.Println` de debug esquecido
- [ ] Erros novos usam `%w` quando o chamador pode inspecioná-los
- [ ] Nenhuma dependência nova sem justificativa em PR description
- [ ] Efeitos de texto de cartas novos documentados em `PLANO.md` se relevantes para etapas futuras
- [ ] `web/dist` recompilado (`task web-build`) se `web/src` mudou
