# pokemon-self-play

> **Boas práticas Go obrigatórias:** siga `docs/go-best-practices.md` em todo código Go escrito neste projeto — erros com `%w`, zero dependências externas, motor single-threaded, `task check` deve passar.

> **Comentários:** não escreva comentários. O código deve ser legível o suficiente para dispensá-los — nomes precisos, funções pequenas, fluxo óbvio. A única exceção são godocs de funções/tipos exportados quando o contrato não é óbvio pela assinatura. Ao tocar qualquer arquivo, apague comentários que já existam (exceto godocs bem escritos de exportados). Nunca documente o "quê" — só o "porquê" oculto, e apenas quando não há outra forma de expressar isso no próprio código.

App em Go para jogar Pokémon TCG sozinho com cartas físicas. Uma câmera aponta para a mesa e enxerga todas as zonas do jogo; o app reconhece cartas e estado da partida, atua como oponente e árbitro.

**Fases planejadas do projeto:**
1. Base de conhecimento das regras (este documento) — concluída.
2. Base de cartas bilíngue + scaffold Go — concluída.
3. Motor de regras em Go: validar ações, manter estado da partida.
4. Visão computacional: reconhecer cartas e mapear zonas físicas da mesa.
5. Oponente automatizado: tomar decisões de jogo pelo lado adversário, incluindo montar o próprio deck (ver seção Construção de deck pelo oponente).

## Estrutura do projeto

```
cmd/import/          CLI que baixa cartas do TCGdex (EN+PT) para data/cards.json
cmd/play/            servidor web da partida contra o bot (serve o build de web/ + /api)
web/                 frontend da partida (Vite + React + TS); build em web/dist,
                     embutido no binário via go:embed (web/embed.go) — dist é commitado
                     para o build Go funcionar sem Node
internal/cards/      modelo canônico de carta, cliente TCGdex, store JSON
internal/deck/       decklist + validação de construção (60 cartas, 4 cópias, ACE SPEC...)
internal/game/       motor de regras: estado, ações de turno, combate, checkup, arbitragem
internal/bot/        oponente: construtor de deck por tipo + piloto de turno heurístico
data/cards.json      base local de cartas (git-ignored; regenerável)
```

Roadmap com todas as etapas e o que está pronto: **`PLANO.md`**.

**Motor de regras (`internal/game`)** — fase 3, implementado:
- Árbitro com estado completo (zonas, prêmios, condições especiais); determinístico por seed.
- Ações validadas: setup/mulligan, energia (1/turno), evolução (com restrições de turno), Item/Suporte/Estádio/Ferramenta (limites), recuo, ataque (Fraqueza ×2 / Resistência −30), nocaute/prêmios/promoção, checkup entre turnos, vitória (prêmios, sem Pokémon, deck-out).
- **Efeitos de texto não são interpretados**: ataques com efeito, Treinadores e Habilidades têm o limite/mecânica aplicados pelo motor e o efeito resolvido manualmente pelos helpers de `arbiter.go` (ApplyDamage, Heal, SetCondition, DrawCards, SwitchActive...). Automatizar efeitos carta a carta é trabalho incremental futuro.
- Prêmios por nocaute: heurística por nome em `PrizeValue` (ex = 2, Mega ex = 3) — TCGdex não expõe a Rule Box.

**Comandos** (via go-task; `task` sem argumento lista tudo — ver `Taskfile.yml`):
- `task play` — partida contra o bot em http://localhost:5173, com reload de front (Vite HMR) e back (watcher reinicia servidor ao mudar .go) (`task play MYTYPE=Grass BOTTYPE=Fire SEED=3`).
- `task import -- <setID> [setID...]` — importa sets (IDs TCGdex, ex.: `me01`, `sve`; lista em `api.tcgdex.net/v2/en/sets`; flag `-standard-only` filtra H/I/J).
- `task check` — build + vet + testes (rodar antes de commitar).
- `task web-build` — recompila o frontend para web/dist (rodar após mudar web/src; commitar o dist).
- `task web` — só o frontend em modo dev (Vite com HMR, proxy /api → :8080); já incluso em `task play`.
- Equivalentes diretos: `go run ./cmd/play ...`, `go run ./cmd/import ...`, `go test ./...`.

**Desenvolvimento (importante para o Claude):** ao trabalhar em mudanças em dev, o usuário já está com os servidores rodando (`task play` + `task web`). **Não iniciar servidores** e **não usar Playwright/browser para validar** — o usuário sempre valida visualmente por conta própria em http://localhost:5173. Após mudanças no frontend, basta garantir que compila (`npx tsc --noEmit` em web/) e avisar que está pronto para validação. **Arquivos temporários** (scripts descartáveis, screenshots, saídas de debug) devem ser **deletados após o uso** — não deixar lixo no repositório.

Nota: campo `legal.standard` dos sets no TCGdex é desatualizado/incorreto — legalidade sempre pelo `regulationMark` da carta (ver `Card.StandardLegal`).

Este documento é a referência canônica de regras para todas as fases. Formato: **Standard** (ver seção Formato e Legalidade).

**Cartas físicas em dois idiomas:** a coleção do usuário mistura cartas em **português (PT-BR)** e **inglês**. Consequências para o sistema:
- O reconhecimento visual precisa identificar a mesma carta em ambas as impressões (nome, texto de ataque e nome de expansão mudam de idioma; número da carta na coleção e layout são iguais).
- A base de cartas deve armazenar nome/texto em EN e PT-BR, indexados pelo mesmo ID canônico. Fonte de dados: **TCGdex API** (`api.tcgdex.net/v2/{en|pt}/...`), que fornece as duas localizações com IDs compartilhados.
- O motor de regras opera sobre o ID canônico — idioma é atributo de exibição/reconhecimento, nunca de lógica.

## Construção de deck pelo oponente (regras de negócio)

O oponente automatizado monta o **próprio deck** antes da partida. Regras:

1. **Fonte das cartas: base de cartas da API (TCGdex).** O deck do oponente é montado a partir do pool completo de cartas legais em `data/cards.json` (importado do TCGdex), sem restrição à coleção física do usuário. O deck do oponente é **virtual**: o app mantém e exibe o estado dele digitalmente; só o lado do jogador humano usa cartas físicas na mesa.
2. **Entrada do usuário: tipos.** Antes da partida, o usuário seleciona o(s) tipo(s) de energia do deck do oponente (ex.: Fire, Water). O construtor filtra o pool por esses tipos (Pokémon do tipo + energias correspondentes + treinadores neutros) e monta o deck de acordo. Sem seleção, o construtor escolhe tipo(s) aleatoriamente (respeitando a seed da regra 5).
3. **Validade obrigatória:** o deck gerado deve passar por todas as regras da seção "Construção do deck" (60 cartas, máx. 4 por nome, 1 ACE SPEC, ≥1 Pokémon Básico) e pela legalidade Standard (`Card.StandardLegal`). O validador de deck é o mesmo usado para o deck do humano — uma única implementação.
4. **Integração com a visão:** a câmera detecta o deck físico do usuário; o app monta o deck do oponente levando em conta a partida configurada (tipos escolhidos + pool da API). Estado do lado do oponente (mão, prêmios, campo) é gerenciado e exibido pelo app, não por cartas físicas.
5. **Determinismo e reprodutibilidade:** dado o mesmo pool, mesmos tipos e mesma seed, o construtor gera o mesmo deck. Facilita teste e depuração. Variedade vem da seed, não de aleatoriedade não controlada.
6. **Estratégia de construção (ordem de preferência):**
   a. **Arquétipos/templates** (padrão): esqueletos de deck (atacante principal + linha de evolução + suportes de consistência + energias) preenchidos por heurística a partir do pool filtrado por tipo. Roda offline, em Go puro, sem custo por partida.
   b. **LLM como camada opcional** (ex.: Claude API): útil só para escolhas "criativas" (escolher arquétipo dentro do tipo, ajustar proporções). Nunca é fonte de verdade — toda saída de LLM passa pelo validador da regra 3. Não é dependência obrigatória: o app funciona 100% offline sem ela.
7. **Sem informação privilegiada:** o construtor pode conhecer a decklist do humano apenas se o usuário optar por isso (modo "counter-deck", off por padrão). Padrão: constrói conhecendo só os tipos selecionados e o pool da API.

---

# Regras do Pokémon TCG — Formato Standard

## 1. Formato e legalidade (vigente em julho/2026)

- Standard 2026: cartas com marca de regulação **H, I ou J** (canto inferior da carta) são legais. Marca **G** rotacionou.
- Rotação em vigor desde **10/abr/2026** (eventos presenciais Play! Pokémon) e **26/mar/2026** (TCG Live).
- Legalidade é pela **marca de regulação**, não pela expansão. Impressões antigas sem marca podem ser usadas se existir versão legal da mesma carta (mesmo nome e mesmo texto funcional).
- Fontes: [anúncio oficial da rotação 2026](https://www.pokemon.com/us/pokemon-news/2026-pokemon-tcg-standard-format-rotation-announcement), [regras Play! Pokémon](https://play.pokemon.com/en-us/resources/rules/), [Bulbapedia 2026-27 Standard](https://bulbapedia.bulbagarden.net/wiki/2026-27_Standard_format_(TCG)).

> Para o motor de regras: legalidade de deck é um filtro de construção, não de jogo. O motor pode aceitar qualquer carta reconhecida e validar legalidade separadamente.

## 2. Construção do deck

- Exatamente **60 cartas**.
- Máximo **4 cópias** de cartas com o mesmo nome (Pokémon e Treinador).
- **Energia básica**: sem limite de cópias.
- Máximo **1 carta ACE SPEC** por deck (qualquer ACE SPEC conta no mesmo limite de 1).
- Máximo **1 Radiant Pokémon** por deck (quando legal no formato).
- O deck precisa conter ao menos 1 Pokémon Básico (regra prática: sem Básico não há mão inicial válida).

## 3. Setup da partida

1. Cumprimento e sorteio (cara ou coroa / par ou ímpar) para decidir quem escolhe se joga primeiro ou segundo.
2. Cada jogador embaralha seu deck e compra **7 cartas**.
3. Cada jogador coloca **1 Pokémon Básico virado para baixo** como Pokémon Ativo. Obrigatório ter ao menos 1 Básico na mão.
   - **Mulligan**: quem não tiver Básico revela a mão, embaralha de volta e compra 7 novas. O oponente pode comprar **1 carta extra por mulligan** do adversário (decide após ver quantos mulligans houve; compra antes de o jogo começar).
4. Cada jogador pode colocar até **5 Pokémon Básicos virados para baixo no Banco**.
5. Cada jogador separa as **6 cartas do topo do deck, viradas para baixo, como Prêmios** (sem olhar).
6. Ambos viram seus Pokémon para cima. Começa o turno do primeiro jogador.

**Restrições do primeiro turno do jogador que começa:**
- **Não pode atacar** no primeiro turno da partida.
- **Não pode jogar carta de Suporte** no seu primeiro turno.
- Pode: ligar Energia, jogar Itens, evoluir não (ver regra de evolução — nenhum Pokémon pode evoluir no primeiro turno em que entrou em jogo, e o turno 1 de cada jogador não permite evoluir Pokémon colocados no setup), usar Habilidades, recuar.
- O jogador que começa **compra carta normalmente** no início do seu primeiro turno.

## 4. Zonas do jogo

| Zona | Inglês | Descrição | Visível? |
|---|---|---|---|
| Pokémon Ativo | Active Pokémon / Active Spot | 1 Pokémon que luta; único que pode atacar e recuar | Face para cima |
| Banco | Bench | Até **5** Pokémon reservas | Face para cima |
| Deck | Deck | Pilha de compra, face para baixo | Não (só contagem) |
| Mão | Hand | Cartas na mão do jogador | Só o dono vê |
| Pilha de descarte | Discard Pile | Cartas descartadas, face para cima | Pública, ordem livre para consulta |
| Prêmios | Prize Cards | 6 cartas face para baixo; pega-se ao nocautear Pokémon inimigo | Não (nem o dono) |
| Estádio | Stadium | Zona compartilhada, no centro; **1 Estádio em jogo por vez** | Face para cima |
| Zona de Perdidos | Lost Zone | Cartas removidas do jogo (não voltam); usada por certas cartas | Face para cima, separada do descarte |

**Anexos ao Pokémon** (ficam fisicamente sob/junto à carta): Energias ligadas, Ferramenta (Pokémon Tool, normalmente 1 por Pokémon), cartas de evolução por baixo, contadores de dano, marcador de condição especial.

> Para a visão computacional: Ativo, Banco, Estádio, descarte e Zona de Perdidos são zonas de leitura direta (face para cima). Deck e Prêmios só exigem contagem/posição. A mão do jogador humano pode ficar fora do campo da câmera ou visível conforme o design escolhido — o oponente automatizado não pode usar informação da mão do humano nas decisões.

## 5. Tipos de carta

### 5.1 Pokémon
- **Estágios**: Básico → Estágio 1 → Estágio 2. Só Básicos entram em jogo diretamente.
- Atributos na carta: HP, tipo (Grass, Fire, Water, Lightning, Psychic, Fighting, Darkness, Metal, Dragon, Colorless), ataques com custo de Energia, Habilidade (Ability, opcional), Fraqueza, Resistência, custo de Recuo.
- **Pokémon ex** (marca de regulação G+): têm "Rule Box"; quando nocauteados, o oponente pega **2 Prêmios**.
- **Tera Pokémon ex**: enquanto no Banco, **não recebem dano** de ataques (efeitos que não são dano ainda se aplicam).
- **Mega Evolution ex** (expansões 2025+): seguem as regras impressas na carta; nocaute concede os Prêmios indicados na carta (algumas valem **3 Prêmios** — sempre ler a Rule Box impressa).

### 5.2 Treinador (Trainer)
| Subtipo | Limite | Quando | Permanência |
|---|---|---|---|
| **Item** | Sem limite por turno | Durante seu turno, antes do ataque | Descarta após o uso |
| **Suporte** (Supporter) | **1 por turno** | Durante seu turno, antes do ataque | Descarta após o uso |
| **Ferramenta** (Pokémon Tool) | Sem limite por turno; **1 ferramenta ligada por Pokémon** | Liga a um Pokémon seu | Permanece ligada; descartada se o Pokémon sair de jogo |
| **Estádio** (Stadium) | **1 por turno** | Substitui o Estádio em jogo (o anterior é descartado); não pode jogar um Estádio com o **mesmo nome** do que já está em jogo | Permanece até ser substituído/descartado |

- **ACE SPEC**: Itens/Ferramentas/Estádios/Energias especiais poderosos; máximo 1 por deck.

### 5.3 Energia
- **Básica**: 1 dos tipos elementares; sem limite no deck; imune a muitos efeitos que afetam "Energia Especial".
- **Especial**: fornece efeitos/energias extras; máximo 4 por nome; conta como Treinador para limites de cópia, mas é jogada como Energia.
- **Ligação**: **1 Energia por turno** da mão a um dos seus Pokémon (Ativo ou Banco). Efeitos de cartas podem ligar Energias adicionais — isso não consome a ligação normal do turno.

## 6. Estrutura do turno

Ordem obrigatória:

1. **Comprar 1 carta** do deck (obrigatório; **se não puder comprar porque o deck acabou, o jogador perde imediatamente** — deck-out).
2. **Fase de ações** (qualquer ordem, qualquer quantidade salvo limites):
   - Colocar Pokémon Básicos no Banco (até completar 5).
   - **Evoluir** Pokémon: colocar Estágio 1 sobre o Básico correspondente, ou Estágio 2 sobre o Estágio 1. Restrições: não evoluir um Pokémon **no turno em que ele entrou em jogo**; não evoluir **no primeiro turno de cada jogador**; não evoluir o mesmo Pokémon duas vezes no mesmo turno (exceto por efeito de carta, ex.: Rare Candy — Básico direto para Estágio 2). Evoluir **remove Condições Especiais** e efeitos aplicados ao Pokémon (dano permanece).
   - Ligar **1 Energia** da mão (Ativo ou Banco).
   - Jogar **Itens** (ilimitado), **1 Suporte**, **1 Estádio**, ligar **Ferramentas**.
   - Usar **Habilidades** (conforme texto de cada uma; muitas são 1x por turno).
   - **Recuar** o Pokémon Ativo: **1 vez por turno**; descartar Energias ligadas iguais ao custo de Recuo; trocar com um Pokémon do Banco. Recuar remove Condições Especiais do Pokémon que recuou. Pokémon **Adormecido ou Paralisado não pode recuar** (por meios normais).
3. **Atacar** (opcional, encerra o turno): declarar 1 ataque do Pokémon Ativo cujo custo de Energia esteja pago (Energia Colorless no custo aceita qualquer tipo). Aplicar dano e efeitos ao alvo. Atacar **encerra o turno imediatamente** — nenhuma outra ação depois.
   - Alternativa: passar o turno sem atacar.
4. **Pokémon Checkup (checagem entre turnos)** — ocorre **entre os turnos**, após o turno de cada jogador (ver seção 8).

## 7. Combate

### 7.1 Cálculo de dano do ataque (ordem)
1. Dano base impresso no ataque.
2. Modificadores por efeitos do atacante ("+X", efeitos de cartas, Ferramentas etc.).
3. **Fraqueza** (Weakness) do defensor: se o tipo do atacante coincide, o dano é multiplicado (**×2** no padrão atual). Aplica só ao Pokémon Ativo defensor.
4. **Resistência** (Resistance) do defensor: subtrai (**−30** no padrão atual). Aplica só ao Pokémon Ativo defensor.
5. Efeitos no defensor que reduzem/previnem dano.
6. Colocar contadores de dano (múltiplos de 10). Fraqueza e Resistência **não se aplicam a dano no Banco** nem a "colocar contadores de dano" (efeitos que dizem "coloque X contadores" ignoram F/R e modificadores).

### 7.2 Nocaute (Knock Out)
- Pokémon com dano total ≥ HP é **Nocauteado**: vai para a pilha de descarte com todas as cartas ligadas (Energias, Ferramentas, evoluções).
- O jogador que nocauteou pega **Prêmios**: 1 (Pokémon comum), 2 (ex/V etc., conforme Rule Box), 3 (alguns Mega ex — conforme carta).
- Se o Ativo foi nocauteado, o dono **promove** um Pokémon do Banco para Ativo. Sem Pokémon no Banco = derrota.
- **Nocautes simultâneos**: resolvem-se ao mesmo tempo; ambos os lados pegam os Prêmios devidos e promovem novos Ativos.

## 8. Condições Especiais (Special Conditions)

Aplicam-se **somente ao Pokémon Ativo**. Removidas quando o Pokémon: recua, evolui, ou vai para o Banco por qualquer efeito.

Marcação física: **Envenenado/Queimado** usam marcadores sobre a carta; **Adormecido** = carta girada 90° anti-horário; **Confuso** = carta de cabeça para baixo; **Paralisado** = carta girada 90° horário. Adormecido/Confuso/Paralisado são mutuamente exclusivos (a mais nova substitui); Envenenado e Queimado coexistem entre si e com as demais.

**Ordem de resolução no Pokémon Checkup (entre turnos):**
1. **Envenenado (Poisoned)**: coloca **1 contador de dano** (10) — alguns efeitos aumentam para mais contadores.
2. **Queimado (Burned)**: coloca **2 contadores de dano** (20); depois joga moeda — **cara: remove a queimadura**; coroa: continua queimado.
3. **Adormecido (Asleep)**: joga moeda — **cara: acorda**; coroa: continua dormindo. Adormecido não pode atacar nem recuar.
4. **Paralisado (Paralyzed)**: não faz checagem com moeda; **remove-se automaticamente no Checkup ao final do turno do próprio jogador** (o Pokémon fica sem atacar/recuar durante um turno completo do dono).

**Confuso (Confused)**: sem efeito no Checkup. Ao **declarar ataque**, joga moeda: cara = ataca normal; **coroa = o ataque falha e o Pokémon coloca 3 contadores de dano em si mesmo** (30). Pode recuar normalmente (recuar cura).

## 9. Vitória e derrota

Um jogador **vence** quando qualquer uma ocorre:
1. Pega seu **último Prêmio** (6 Prêmios coletados).
2. O oponente **não tem Pokémon em jogo** para promover como Ativo.
3. O oponente **não pode comprar carta** no início do turno dele (deck-out).

**Condições simultâneas / Sudden Death:** se ambos os jogadores atingirem uma condição de vitória ao mesmo tempo (ex.: nocaute duplo em que ambos pegam o último Prêmio), joga-se **Sudden Death**: nova partida com **1 Prêmio** em vez de 6, mesmas regras de setup. Se as condições simultâneas forem *diferentes*, há ordem de precedência: quem cumpre mais condições vence; condições equivalentes → Sudden Death.

> Para jogo solo: empate real é raro; implementar Sudden Death como partida reduzida de 1 Prêmio é suficiente.

## 10. Regras gerais de arbitragem relevantes para o motor

- **Efeitos de carta vencem regras gerais**: se o texto da carta contradiz uma regra, vale a carta.
- **"Faça o máximo possível"**: instruções de carta são executadas ao máximo possível; partes impossíveis são ignoradas (mas há cartas que exigem custo integral — ler texto).
- Buscar no deck (**search**) = deck é embaralhado depois; buscas "por um Pokémon Básico" podem falhar propositalmente (informação oculta), mas cartas que nomeiam critério verificável exigem revelar.
- **Informação pública**: contagem de cartas na mão, deck, descarte e Prêmios de ambos é pública. Conteúdo do descarte é público e consultável a qualquer momento.
- Contadores de dano ficam no Pokémon mesmo ao ir para o Banco; dano nunca "cura" sozinho.
- Efeitos que dizem "dano" sofrem F/R e modificadores; efeitos que dizem "coloque contadores de dano" não.
- Uma vez iniciada a resolução de um ataque/carta, resolve-se por completo antes de qualquer outra ação.

## 11. Glossário de zonas e termos (EN ↔ PT)

| Inglês | Português | Nota para mapeamento da câmera |
|---|---|---|
| Active Spot / Active Pokémon | Pokémon Ativo | 1 slot por jogador, frente da mesa |
| Bench | Banco | Até 5 slots por jogador, atrás do Ativo |
| Deck | Baralho / Deck | Pilha face para baixo |
| Hand | Mão | Fora da mesa ou área designada |
| Discard Pile | Pilha de Descarte | Face para cima, ao lado do deck |
| Prize Cards | Cartas de Prêmio | 6 cartas face para baixo, geralmente 2 colunas de 3 |
| Stadium | Estádio | Slot único compartilhado, centro da mesa |
| Lost Zone | Zona de Perdidos | Área separada, face para cima |
| Attack | Ataque | — |
| Ability | Habilidade | — |
| Weakness / Resistance | Fraqueza / Resistência | Rodapé da carta de Pokémon |
| Retreat Cost | Custo de Recuo | Rodapé da carta |
| Knock Out (KO) | Nocaute | — |
| Damage counter | Contador de dano | Fichas de 10/50/100 sobre a carta |
| Special Condition | Condição Especial | Orientação da carta + marcadores |
| Poisoned / Burned | Envenenado / Queimado | Marcadores específicos |
| Asleep / Confused / Paralyzed | Adormecido / Confuso / Paralisado | Rotação da carta: 90° anti-horário / 180° / 90° horário |
| Supporter / Item / Tool / Stadium | Suporte / Item / Ferramenta / Estádio | Subtipo impresso na carta de Treinador |
| Basic / Special Energy | Energia Básica / Especial | — |
| Evolution (Stage 1 / Stage 2) | Evolução (Estágio 1 / 2) | Cartas empilhadas — câmera vê só a do topo |
| Mulligan | Mulligan | Só ocorre no setup |
| Prize (take a Prize) | Pegar Prêmio | Carta sai da zona de Prêmios para a mão |
| Deck-out | Deck-out | Derrota por não poder comprar |
| Regulation Mark | Marca de Regulação | Letra no canto inferior esquerdo da carta |
| Rule Box | Rule Box | Caixa de regra em ex/V etc. — define Prêmios extras |

## 12. Fontes

- [Regras e formatos oficiais Play! Pokémon](https://play.pokemon.com/en-us/resources/rules/) — rulebook oficial e documentos de penalidade.
- [Anúncio da rotação Standard 2026](https://www.pokemon.com/us/pokemon-news/2026-pokemon-tcg-standard-format-rotation-announcement) — marcas H/I/J, datas de vigência.
- [Bulbapedia — 2026-27 Standard format](https://bulbapedia.bulbagarden.net/wiki/2026-27_Standard_format_(TCG)) — lista de expansões legais.
- Documento vivo: atualizar seção 1 a cada rotação anual (tipicamente abril).
