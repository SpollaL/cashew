package main

import (
	"github.com/spolla-l/cashew/internal/domain"
	"github.com/spolla-l/cashew/internal/parser"
	"github.com/spolla-l/cashew/internal/rules"
	"github.com/spolla-l/cashew/internal/server"
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	rulesPath := flag.String("rules", "rules.toml", "path to rules.toml")
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "usage: cashew-server [flags] <file> [file2 ...]")
		os.Exit(1)
	}

	parsers := []parser.Parser{parser.Revolut{}, parser.BBVA{}}

	var allTxs []domain.Transaction
	for _, path := range files {
		txs, err := parser.Load(path, parsers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading %s: %v\n", path, err)
			os.Exit(1)
		}
		allTxs = append(allTxs, txs...)
	}
	allTxs = parser.Deduplicate(allTxs)

	rulesList, buckets, err := rules.Load(*rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading rules: %v\n", err)
		os.Exit(1)
	}

	srv := server.New(allTxs, rulesList, buckets, *rulesPath)

	fmt.Printf("cashew running at http://localhost%s\n", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
