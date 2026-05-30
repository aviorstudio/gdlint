package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aviorstudio/gdlint/src/core"
	"github.com/aviorstudio/gdlint/src/models"
	"github.com/aviorstudio/gdlint/src/operations"
)

var (
	fixFlag       = flag.Bool("fix", false, "Remove all unused code and clean files")
	warnFlag      = flag.Bool("warn", false, "Show warnings in addition to errors")
	debtFlag      = flag.Bool("debt", false, "Show [TECH DEBT] comments")
	benchmarkFlag = flag.Bool("benchmark", false, "Show performance timing")
	version       = "dev"
	commit        = "unknown"
	date          = "unknown"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s init\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s version\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nGDScript Linter - Analyzes and cleans GDScript code\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  init           Create default gdlint.json in the current Godot project\n")
		fmt.Fprintf(os.Stderr, "  version        Print version information\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s              # Check for issues and list all\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -fix         # Remove all unused code\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -warn        # Show warnings\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s init         # Create gdlint.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s version      # Print version information\n", os.Args[0])
	}

	flag.Parse()
	command, err := parseCommand()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}
	if command == "version" {
		printVersion()
		os.Exit(0)
	}

	rootDir, err := validateProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error validating project root: %v\n", err)
		os.Exit(1)
	}

	if command == "init" {
		if err := createDefaultConfig(rootDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Created default configuration file: gdlint.json")
		os.Exit(0)
	}

	startTime := time.Now()

	config, err := loadConfig(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if *fixFlag {
		runFixMode(rootDir, config)
	} else if *debtFlag {
		showTechDebt(rootDir, config)
	} else {
		runAnalysisMode(rootDir, config)
	}

	if *benchmarkFlag {
		fmt.Printf("\nExecution time: %.2fs\n", time.Since(startTime).Seconds())
	}
}

func parseCommand() (string, error) {
	if flag.NArg() == 0 {
		return "", nil
	}
	if flag.NArg() > 1 {
		return "", fmt.Errorf("expected at most one command")
	}

	command := flag.Arg(0)
	if command != "init" && command != "version" {
		return "", fmt.Errorf("unknown command: %s", command)
	}
	return command, nil
}

func printVersion() {
	fmt.Printf("gdlint %s (%s, %s)\n", version, commit, date)
}

func validateProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	projectFile := filepath.Join(cwd, "project.godot")
	info, err := os.Stat(projectFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("current directory must be a Godot project root containing project.godot")
		}
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("project.godot is a directory, expected a file")
	}

	return cwd, nil
}

func loadConfig(rootDir string) (*models.LintConfig, error) {
	config := models.NewDefaultConfig()
	configPath := filepath.Join(rootDir, "gdlint.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return config, nil
}

func createDefaultConfig(rootDir string) error {
	configPath := filepath.Join(rootDir, "gdlint.json")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config already exists: %s", configPath)
	} else if !os.IsNotExist(err) {
		return err
	}

	config := models.NewDefaultConfig()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func runAnalysisMode(rootDir string, config *models.LintConfig) {
	analyzer := core.NewAnalyzer(rootDir, config)
	results, err := analyzer.Analyze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during analysis: %v\n", err)
		os.Exit(1)
	}

	display := operations.NewDisplay()

	hasErrors := display.ShowResults(results, *warnFlag, 0)

	if hasErrors {
		os.Exit(1)
	}
}

func runFixMode(rootDir string, config *models.LintConfig) {
	iteration := 0
	maxIterations := 10

	for iteration < maxIterations {
		iteration++
		fmt.Printf("\n=== Fix iteration %d ===\n", iteration)

		analyzer := core.NewAnalyzer(rootDir, config)
		results, err := analyzer.Analyze()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during analysis: %v\n", err)
			os.Exit(1)
		}

		if !results.HasErrors() {
			fmt.Println("✅ No issues found - codebase is clean!")
			return
		}

		remover := operations.NewRemover(rootDir)
		if err := remover.RemoveAll(results); err != nil {
			fmt.Fprintf(os.Stderr, "Error during removal: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "Warning: Reached maximum iterations (%d) - there may still be issues\n", maxIterations)
}

func showTechDebt(rootDir string, config *models.LintConfig) {
	scanner := operations.NewScanner(rootDir)
	comments, err := scanner.FindTechDebtComments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning for tech debt: %v\n", err)
		os.Exit(1)
	}

	if len(comments) == 0 {
		fmt.Println("No [TECH DEBT] comments found")
		return
	}

	fmt.Printf("Found %d [TECH DEBT] comments:\n\n", len(comments))
	for _, comment := range comments {
		fmt.Printf("%s:%d: %s\n", comment.File, comment.Line, comment.Text)
	}
}
