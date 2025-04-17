package app

import (
	"fmt"
	"strings"

	"github.com/byte/gohttpprobe/internal/probe"
	"github.com/spf13/cobra"
)

// Version template
const versionTemplate = "{{banner}}{{if .Version}}{{.Version}}{{end}}\n"

var (
	rootCmd = &cobra.Command{
		Use:   "ghp",
		Short: "HTTP Methods Tester",
		Long: "GoHTTPProbe is a tool for testing HTTP methods against URLs. " +
			"It can be used to find HTTP verb tampering vulnerabilities and \"dangerous\" HTTP methods.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config from flags
			config := probe.Config{
				URL:         targetURL,
				Verbose:     verbose,
				Quiet:       quiet,
				Insecure:    insecure,
				FollowRedir: followRedir,
				SafeOnly:    safeOnly,
				Wordlist:    wordlist,
				Threads:     threads,
				JSONFile:    jsonFile,
				Proxy:       proxy,
				Cookies:     cookies,
				Headers:     headers,
				InputFile:   inputFile,
				CookieJar:   cookieJar,
				Timeout:     timeout,
			}

			// Run the probe
			return probe.Run(config)
		},
		Version: "0.0.1",
	}

	// Command line flags
	targetURL   string
	verbose     bool
	quiet       bool
	insecure    bool
	followRedir bool
	safeOnly    bool
	wordlist    string
	threads     int
	jsonFile    string
	proxy       string
	cookies     string
	headers     []string
	inputFile   string
	cookieJar   string
	timeout     int
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Print the version with the banner
	cobra.AddTemplateFunc("banner", func() string {
		return getBannerText(rootCmd.Version) + "\n"
	})

	// Set both help and usage functions to use help
	rootCmd.SetHelpFunc(help)
	rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		help(cmd, nil)
		return nil
	})

	// Set version template
	rootCmd.SetVersionTemplate(versionTemplate)

	// Register all flags
	for _, category := range flagCategories {
		for _, flag := range category.Flags {
			flag.Register(rootCmd)
		}
	}

	// Mark required flags
	rootCmd.MarkFlagsMutuallyExclusive("url", "input")
	rootCmd.MarkFlagsOneRequired("url", "input")
}

// help is a custom help function that displays flags in organized categories
func help(cmd *cobra.Command, args []string) {
	// Print banner and usage
	fmt.Println(getBannerText(cmd.Version))
	fmt.Printf("\nUsage: %s\n\n", cmd.UseLine())
	fmt.Println("Options:")

	// Print each category with its flags
	for _, category := range flagCategories {
		fmt.Printf("  # %s:\n", category.Name)

		for _, flagDef := range category.Flags {
			// Format the flag information
			name := "--" + flagDef.GetLongName()
			shorthand := flagDef.GetShortName()
			if shorthand != "" {
				name = "-" + shorthand + ", " + name
			}

			fmt.Printf("  %-24s        %s\n", name, flagDef.GetDescription())
		}
		fmt.Println()
	}
}

func normalizeHeaderFlags(headers []string) []string {
	var normalized []string
	for _, h := range headers {
		// Handle comma-separated headers
		if strings.Contains(h, ",") {
			parts := strings.Split(h, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					normalized = append(normalized, trimmed)
				}
			}
		} else {
			normalized = append(normalized, h)
		}
	}
	return normalized
}

func getBannerText(version string) string {
	return "[~] GoHTTPProbe - HTTP Methods Tester v" + version
}
