package main

import (
	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/parser"
	"github.com/SpollaL/cashew/internal/rules"
	"github.com/SpollaL/cashew/internal/server"
	"flag"
	"fmt"
	"net/http"
	"os"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	rulesPath := flag.String("rules", "rules.toml", "path to rules.toml")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		fmt.Println(version)
		return
	}

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
