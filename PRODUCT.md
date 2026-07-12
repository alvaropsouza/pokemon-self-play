# Product

## Register

product

## Users

Um único usuário: o dono do projeto, jogando Pokémon TCG sozinho em casa com cartas físicas na mesa. O app é a mesa digital do oponente (bot) e o árbitro da partida. Contexto: sessão de jogo relaxada, tela de laptop ou monitor ao lado da mesa física, luz ambiente variável.

## Product Purpose

Interface web da partida contra o bot (`cmd/play` + `web/`). Mostra o tabuleiro completo (duas metades, pilhas, prêmios, mão), permite as ações de turno do jogador humano e a arbitragem manual de efeitos que o motor não interpreta. Sucesso = o usuário consegue jogar uma partida inteira sem confusão sobre estado ou regras.

## Brand Personality

Mesa de jogo, sóbria, funcional. Remete a um tapete (playmat) de TCG real: zonas nomeadas, pilhas físicas, Pokébola como motivo discreto. Sem gamificação artificial, sem mascotes, sem tom infantil — a diversão vem das cartas, não do chrome da UI.

## Anti-references

- UI de jogo mobile free-to-play (brilhos, badges, confete).
- Dashboards SaaS genéricos (cards idênticos, hero metrics).
- Emoji como ícone de interface.

## Design Principles

1. **A mesa é a interface.** O layout espelha a mesa física; zonas do jogo têm nome e posição fixa, como num playmat.
2. **Estado sempre visível.** Turno, fase, vez, contagens de deck/mão/prêmios legíveis o tempo todo; nada escondido atrás de cliques.
3. **Ações contextuais.** Só aparecem os botões válidos para a fase e a seleção atual — o app é também o árbitro.
4. **Densidade sem estouro.** Tudo cabe em 100vh; carta escala com a altura da tela.
5. **Motion só para estado.** Animações curtas marcam entrada de carta/energia; nada decorativo.

## Accessibility & Inclusion

Sem requisito WCAG formal (uso pessoal), mas básicos valem: contraste legível no tema escuro, `prefers-reduced-motion` respeitado, foco visível em controles.
