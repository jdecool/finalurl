package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jdecool/finalurl/checker"
)

const (
	version = "1.0.0"
	banner  = `
┌─────────────────────────────────────┐
│            FinalURL v%s          │
│     URL Redirection Checker         │
└─────────────────────────────────────┘
`
)

var (
	showRedirect = flag.Bool("show-redirect", false, "Show all redirections in the flow")
	showVersion  = flag.Bool("version", false, "Show version information")
	verbose      = flag.Bool("verbose", false, "Enable verbose output")
	noRobots     = flag.Bool("no-robots", false, "Skip robots.txt checking")
	timeout      = flag.Int("timeout", 30, "Request timeout in seconds")
	help         = flag.Bool("help", false, "Show help information")
)

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *showVersion {
		fmt.Printf("FinalURL version %s\n", version)
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf(banner, version)
		fmt.Println("Error: No URL provided")
		fmt.Println("Usage: finalurl [options] <URL>")
		fmt.Println("Use -help for more information")
		os.Exit(1)
	}

	url := args[0]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	if *verbose {
		fmt.Printf(banner, version)
		fmt.Printf("🔍 Analyzing URL: %s\n", url)
		fmt.Printf("⚙️  Settings: robots.txt=%t, timeout=%ds\n", !*noRobots, *timeout)
		fmt.Println(strings.Repeat("─", 50))
	}

	checker := &checker.Checker{
		CheckRobotTxt: !*noRobots,
	}

	start := time.Now()
	flow, err := checker.GetRedirections(url)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("⏱️  Analysis completed in %v\n", duration)
		fmt.Println(strings.Repeat("─", 50))
	}

	if *showRedirect && len(flow.Redirections) > 0 {
		fmt.Println("🔄 Redirection Flow:")
		fmt.Printf("   Original: %s\n", flow.OriginalURL)

		for i, redirect := range flow.Redirections {
			fmt.Printf("   Step %d:   %s (HTTP %d)\n", i+1, redirect.URL.String(), redirect.StatusCode)
		}

		if flow.FinalResponse != nil {
			fmt.Printf("   Final:    %s (HTTP %d)\n", flow.FinalResponse.URL.String(), flow.FinalResponse.StatusCode)
		}
		fmt.Println()
	}

	if flow.FinalResponse != nil {
		if *verbose {
			fmt.Printf("🎯 Final URL: %s\n", flow.FinalResponse.URL.String())
			fmt.Printf("📊 Status Code: %d\n", flow.FinalResponse.StatusCode)
			fmt.Printf("🔢 Total Redirects: %d\n", len(flow.Redirections))
		} else {
			fmt.Println(flow.FinalResponse.URL.String())
		}
	} else {
		fmt.Println("❌ Could not determine final URL")
		os.Exit(1)
	}

	if *verbose {
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println("✅ Analysis complete!")
	}
}

func showHelp() {
	fmt.Printf(banner, version)
	fmt.Println("A simple tool to get the final destination of a URL after following redirects.")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  finalurl [options] <URL>")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -show-redirect    Show all redirections in the flow")
	fmt.Println("  -verbose          Enable verbose output with detailed information")
	fmt.Println("  -no-robots        Skip robots.txt checking")
	fmt.Println("  -timeout <sec>    Request timeout in seconds (default: 30)")
	fmt.Println("  -version          Show version information")
	fmt.Println("  -help             Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  finalurl https://t.co/HFmwmM31kM")
	fmt.Println("  finalurl -show-redirect -verbose https://bit.ly/example")
	fmt.Println("  finalurl -no-robots https://short.link/abc123")
	fmt.Println()
	fmt.Println("OUTPUT:")
	fmt.Println("  By default, only the final URL is printed to stdout.")
	fmt.Println("  Use -verbose for detailed analysis information.")
	fmt.Println("  Use -show-redirect to see the complete redirection chain.")
}
