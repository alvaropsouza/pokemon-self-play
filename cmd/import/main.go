// Comando import baixa cartas do TCGdex (EN + PT-BR) e grava/atualiza a
// base local em data/cards.json.
//
// Uso:
//
//	go run ./cmd/import <setID> [setID...]      importa sets específicos
//	go run ./cmd/import -standard-only <setID>  só cartas com marca H/I/J
//
// IDs de set são os do TCGdex (ex.: sv10, me01, me02). Lista completa:
// https://api.tcgdex.net/v2/en/sets
package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/alvaropsouza/pokemon-self-play/internal/cards"
)

const dbPath = "data/cards.json"

func main() {
	standardOnly := flag.Bool("standard-only", false, "importar apenas cartas legais no Standard (marcas H/I/J)")
	workers := flag.Int("workers", 8, "requisições concorrentes")
	flag.Parse()

	setIDs := flag.Args()
	anyFailed := false
	if len(setIDs) == 0 {
		fmt.Fprintln(os.Stderr, "uso: import [-standard-only] <setID> [setID...]")
		os.Exit(2)
	}

	store, err := cards.Load(dbPath)
	if err != nil {
		fatal(err)
	}
	client := cards.NewTCGdexClient()

	for _, setID := range setIDs {
		ids, err := client.SetCardIDs(setID)
		if err != nil {
			fatal(fmt.Errorf("set %s: %w", setID, err))
		}
		fmt.Printf("set %s: %d cartas\n", setID, len(ids))

		var (
			mu       sync.Mutex
			wg       sync.WaitGroup
			sem      = make(chan struct{}, *workers)
			imported int
			skipped  int
			failed   int
		)
		for _, id := range ids {
			wg.Add(1)
			sem <- struct{}{}
			go func(id string) {
				defer wg.Done()
				defer func() { <-sem }()
				card, err := client.FetchCard(id)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					failed++
					fmt.Fprintf(os.Stderr, "  erro %s: %v\n", id, err)
					return
				}
				if *standardOnly && !card.StandardLegal() {
					skipped++
					return
				}
				store.Put(card)
				imported++
			}(id)
		}
		wg.Wait()
		fmt.Printf("  importadas %d, puladas %d, falhas %d\n", imported, skipped, failed)
		if failed > 0 {
			anyFailed = true
		}
	}

	if err := store.Save(dbPath); err != nil {
		fatal(err)
	}
	fmt.Printf("base salva em %s (%d cartas, sets: %v)\n", dbPath, len(store.Cards), store.SetIDs())
	if anyFailed {
		os.Exit(1)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
