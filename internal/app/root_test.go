package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func executeCommand(root *cobra.Command, args ...string) (string, error) {
	// Save and restore original stdout/stderr
	oldOut := root.OutOrStdout()
	oldErr := root.ErrOrStderr()
	
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	
	// Restore after the test
	defer func() {
		root.SetOut(oldOut)
		root.SetErr(oldErr)
	}()

	err := root.Execute()
	return buf.String(), err
}

func TestRootCommandValidation(t *testing.T) {
	// Create a test root command that doesn't actually execute the probe
	testRootCmd := &cobra.Command{
		Use:   "gohttpprobe",
		Short: "Test root command",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Just validate that one of the required flags is provided
			url := cmd.Flags().Lookup("url").Value.String()
			input := cmd.Flags().Lookup("input").Value.String()
			if url == "" && input == "" {
				return cmd.Help()
			}
			return nil
		},
	}
	
	// Add the required flags
	testRootCmd.Flags().String("url", "", "Target URL")
	testRootCmd.Flags().String("input", "", "File with URLs")
	testRootCmd.MarkFlagsOneRequired("url", "input")

	// Test without required flags
	_, err := executeCommand(testRootCmd)
	if err == nil {
		t.Error("Expected an error when URL flag is missing, but got none")
	}

	// Test with URL flag
	_, err = executeCommand(testRootCmd, "--url", "http://example.com")
	if err != nil {
		t.Errorf("Unexpected error when running with URL flag: %v", err)
	}
	
	// Test with input flag (in a separate command execution)
	testInputCmd := &cobra.Command{
		Use:   "gohttpprobe",
		Short: "Test root command",
		RunE: func(cmd *cobra.Command, args []string) error {
			input := cmd.Flags().Lookup("input").Value.String()
			if input == "" {
				return cmd.Help()
			}
			return nil
		},
	}
	testInputCmd.Flags().String("input", "", "File with URLs")
	
	_, err = executeCommand(testInputCmd, "--input", "urls.txt")
	if err != nil {
		t.Errorf("Unexpected error when running with input flag: %v", err)
	}
}

func TestRootCommandFlags(t *testing.T) {
	// Verify that all required flags are defined
	requiredFlags := []string{"url", "input"}
	for _, flag := range requiredFlags {
		if rootCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Required flag %q not defined", flag)
		}
	}

	// Verify that the common flags are defined
	commonFlags := []string{
		"verbose", "quiet", "insecure", "follow", "safe-only",
		"methods", "concurrent", "output", "proxy", "cookies",
		"header", "cookie-jar", "user-agent",
	}

	for _, flag := range commonFlags {
		if rootCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Common flag %q not defined", flag)
		}
	}
}

func TestVersionFlag(t *testing.T) {
	// Test the version flag directly on rootCmd
	output, err := executeCommand(rootCmd, "--version")
	if err != nil {
		t.Errorf("Unexpected error when running version command: %v", err)
	}

	if !strings.Contains(output, "GoHTTPProbe") {
		t.Errorf("Version output doesn't contain expected banner: %s", output)
	}
}