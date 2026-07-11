# Plano de desenvolvimento — pokemon-self-play

Roadmap completo do projeto, do estado atual até jogar com cartas físicas e câmera.
Marcar `[x]` conforme concluir. Regras canônicas e decisões de arquitetura: ver `CLAUDE.md`.

## Etapa 0 — Fundações ✅

- [x] Base de conhecimento das regras (CLAUDE.md, formato Standard 2026).
- [x] Modelo canônico de carta bilíngue (EN/PT) — `internal/cards`.
- [x] Importador TCGdex (`cmd/import`), base local `data/cards.json`.
- [x] Set de energias básicas importado (`sve`).

## Etapa 1 — Motor de regras ✅

- [x] Estado completo da partida: zonas, prêmios, condições especiais, estádio (`internal/game`).
- [x] Setup com mulligan, restrições de primeiro turno, limites por turno.
- [x] Combate: custo de energia, Fraqueza ×2 / Resistência −30, nocaute, prêmios, promoção.
- [x] Pokémon Checkup entre turnos (veneno, queimadura, sono, paralisia).
- [x] Condições de vitória (prêmios, sem Pokémon, deck-out) + Sudden Death sinalizado.
- [x] Helpers de arbitragem manual para efeitos de texto (`arbiter.go`).
- [x] Validador de deck (`internal/deck`): 60 cartas, 4 cópias, ACE SPEC, legalidade H/I/J.
- [x] Testes do motor e do validador.

## Etapa 2 — Bot + interface jogável ✅

Objetivo: jogar uma partida completa contra o bot no navegador, tudo virtual.
É a interface que acompanha o desenvolvimento do projeto daqui em diante.

Jogar: `go run ./cmd/play -mytype Fire -bottype Water -seed 7` → http://localhost:8080

- [x] Construtor de deck do bot (`internal/bot`): filtra pool por tipo(s) escolhido(s),
      monta linhas de evolução + energias, determinístico por seed, valida com `internal/deck`.
- [x] Piloto de turno do bot: heurística simples — baixar básicos, evoluir, ligar energia,
      atacar com maior dano pago, promover melhor opção após nocaute.
- [x] Servidor web (`cmd/play`): estado da partida via JSON, ações via POST,
      bot joga automaticamente no turno dele.
- [x] UI no navegador: mesa com as duas metades (imagens das cartas via TCGdex),
      mão do jogador clicável, log da partida, painel de arbitragem manual
      (dano, cura, condição, compra, troca) para efeitos de carta.
- [x] Ocultação de informação: mão/deck/prêmios do bot aparecem só como contagem.
- [x] Teste de fumaça: partidas bot vs bot completas terminam sem travar.

## Etapa 3 — Qualidade de jogo

- [ ] Efeitos automatizados das cartas mais comuns (motor de efeitos por ID de carta):
      compra ("draw N"), busca no deck, troca de Ativo, cura, dano no banco.
      Começar pelos Treinadores de consistência do pool importado.
- [ ] Bot usa Treinadores automatizados (hoje o deck dele evita cartas de efeito).
- [ ] Bot melhor: avaliação de alvo (Boss's Orders), gestão de energia multi-Pokémon,
      decidir recuo, contar prêmios.
- [ ] Importar mais sets Standard (H/I/J) para ampliar o pool.
- [ ] Persistência de partida (salvar/retomar estado em JSON).
- [ ] Modo "counter-deck" opcional (bot conhece a decklist do humano).

## Etapa 4 — Cartas físicas do jogador

- [ ] Inventário da coleção física (carta → quantidade) em `data/`.
- [ ] Registro do deck físico do humano na UI (decklist digitada/importada e validada).
- [ ] Modo espelho: humano joga com cartas físicas na mesa e reflete as ações na UI
      (a UI vira o árbitro; câmera ainda não entra).

## Etapa 5 — Visão computacional

- [ ] Calibração das zonas da mesa (Ativo, Banco, descarte, estádio, prêmios por posição).
- [ ] Reconhecimento de carta nas duas línguas (hash de imagem / matching com base TCGdex).
- [ ] Leitura contínua: detectar mudanças de zona e traduzir em ações do motor.
- [ ] Reconciliação: divergência entre mesa física e estado do motor gera alerta na UI.

## Etapa 6 — Refinamento do oponente

- [ ] Camada opcional de LLM (Claude API) para escolha de arquétipo/decisões criativas —
      sempre validada pelo motor (nunca fonte de verdade; app funciona offline sem ela).
- [ ] Dificuldades do bot (agressivo/consistente/aleatório).
- [ ] Sudden Death jogável (partida reduzida de 1 prêmio automática).

## Decisões já tomadas (resumo)

- Oponente é bot; lado dele é 100% virtual (deck do pool da API, estado exibido na UI).
- Efeitos de texto de carta: arbitragem manual primeiro, automação incremental depois.
- Determinismo por seed em tudo (deck do bot, embaralhamento, moedas).
- Um único validador de deck para humano e bot.
- Legalidade sempre pelo `regulationMark` da carta; energia básica sempre legal.
