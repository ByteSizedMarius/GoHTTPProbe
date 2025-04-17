package app

var flagCategories = []struct {
	Name  string
	Flags []Flag
}{
	{
		Name: "Target selection",
		Flags: []Flag{
			StringFlag{
				BaseFlag: BaseFlag{"url", "u", "Target URL (e.g., example.com:port/path)"},
				Default:  "",
				Target:   &targetURL,
			},
			StringFlag{
				BaseFlag: BaseFlag{"input", "i", "Read target URLs from a file (one per line)"},
				Default:  "",
				Target:   &inputFile,
			},
		},
	},
	{
		Name: "Connection options",
		Flags: []Flag{
			BoolFlag{
				BaseFlag: BaseFlag{"insecure", "k", "Allow insecure server connections (skip SSL verification)"},
				Default:  false,
				Target:   &insecure,
			},
			BoolFlag{
				BaseFlag: BaseFlag{"follow", "f", "Follow redirects"},
				Default:  false,
				Target:   &followRedir,
			},
			StringFlag{
				BaseFlag: BaseFlag{"proxy", "p", "Use proxy for connections (e.g., http://localhost:8080)"},
				Default:  "",
				Target:   &proxy,
			},
			IntFlag{
				BaseFlag: BaseFlag{"concurrent", "n", "Number of concurrent requests (default: 5)"},
				Default:  5,
				Target:   &threads,
			},
			IntFlag{
				BaseFlag: BaseFlag{"timeout", "t", "Timeout in seconds for HTTP requests (default: 10)"},
				Default:  10,
				Target:   &timeout,
			},
		},
	},
	{
		Name: "Request customization",
		Flags: []Flag{
			StringArrayFlag{
				BaseFlag: BaseFlag{"header", "H", "Headers to include (e.g., -H \"User-Agent: test\" or -H headers.txt)"},
				Default:  []string{},
				Target:   &headers,
			},
			StringFlag{
				BaseFlag: BaseFlag{"cookies", "b", "Cookies to use (e.g., -b \"session=abc\" or -b cookies.txt)"},
				Default:  "",
				Target:   &cookies,
			},
			StringFlag{
				BaseFlag: BaseFlag{"cookie-jar", "c", "Write received cookies to specified file"},
				Default:  "",
				Target:   &cookieJar,
			},
			StringPFlag{
				BaseFlag: BaseFlag{"user-agent", "A", "User-Agent string to send"},
				Default:  "",
			},
		},
	},
	{
		Name: "Method testing options",
		Flags: []Flag{
			BoolFlag{
				BaseFlag: BaseFlag{"safe-only", "s", "Only test safe methods (exclude PUT, DELETE, etc.)"},
				Default:  false,
				Target:   &safeOnly,
			},
			StringFlag{
				BaseFlag: BaseFlag{"methods", "m", "Custom HTTP methods wordlist file"},
				Default:  "",
				Target:   &wordlist,
			},
		},
	},
	{
		Name: "Output control",
		Flags: []Flag{
			BoolFlag{
				BaseFlag: BaseFlag{"verbose", "v", "Enable verbose output"},
				Default:  false,
				Target:   &verbose,
			},
			BoolFlag{
				BaseFlag: BaseFlag{"quiet", "q", "Show no information at all"},
				Default:  false,
				Target:   &quiet,
			},
			StringFlag{
				BaseFlag: BaseFlag{"output", "o", "Save results to specified JSON file"},
				Default:  "",
				Target:   &jsonFile,
			},
		},
	},
}
