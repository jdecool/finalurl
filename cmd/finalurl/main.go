package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jdecool/finalurl/checker"
)

const (
	banner = `finalurl
========

A very simple tool to get the final path of an URL
`
)

var (
	displayRedirect bool
)

func main() {
	flag.BoolVar(&displayRedirect, "show-redirect", false, "Show redirections")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprint(os.Stderr, fmt.Sprintf(banner))
		flag.PrintDefaults()

		fmt.Fprintln(os.Stderr, "\nmissing URL")
		os.Exit(1)
	}

	url := flag.Arg(0)

	flow, err := checker.GetRedirections(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if displayRedirect {
		for _, resp := range flow.Redirections {
			fmt.Fprintln(os.Stdout, "-->", resp.URL, "(", resp.StatusCode, ")")
		}

		fmt.Println("")
	}

	fmt.Fprintln(os.Stdout, "**Final URL:**", flow.FinalResponse.URL, "(", flow.FinalResponse.StatusCode, ")")
}
