package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SpollaL/cashew/internal/config"
	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/ledger"
	"github.com/SpollaL/cashew/internal/parser"
	"github.com/SpollaL/cashew/internal/rules"
	"github.com/SpollaL/cashew/internal/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "rules" {
		path, err := config.RulesPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// Bootstrap the file with default buckets if it doesn't exist yet,
		// so editors that can't create files don't fail on first run.
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			if _, _, loadErr := rules.Load(path); loadErr != nil {
				fmt.Fprintln(os.Stderr, loadErr)
				os.Exit(1)
			}
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		cmd := exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	model := flag.String("model", "hf.co/Qwen/Qwen3-4B-GGUF:Q4_K_M", "Ollama model to use for chat (e.g. gemma3, llama3.2)")
	debug := flag.Bool("debug", false, "show LLM roundtrip details in the chat view")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		fmt.Println(version)
		return
	}

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cashew [-model <name>] <file> [file2 ...]")
		os.Exit(1)
	}

	rulesPath, err := config.RulesPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving rules path:", err)
		os.Exit(1)
	}

	parsers := []parser.Parser{
		parser.Revolut{},
		parser.BBVA{},
	}

	var allTxs []domain.Transaction
	for _, path := range flag.Args() {
		txs, err := parser.Load(path, parsers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading %s: %v\n", path, err)
			os.Exit(1)
		}
		allTxs = append(allTxs, txs...)
	}

	allTxs = parser.Deduplicate(allTxs)

	rulesList, buckets, err := rules.Load(rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading rules: %v\n", err)
		os.Exit(1)
	}

	rawTxs := allTxs
	allTxs = rules.Apply(allTxs, rulesList)
	l := ledger.New(allTxs)

	app := tui.New(l, rawTxs, rulesList, buckets, rulesPath, *model, *debug)
	p := tea.NewProgram(app, tea.WithAltScreen())
	app.SetProgram(p)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
