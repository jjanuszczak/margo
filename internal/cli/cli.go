package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"margo/internal/archetype"
	"margo/internal/clean"
	"margo/internal/config"
	"margo/internal/content"
	"margo/internal/deck"
	"margo/internal/diagnostics"
	"margo/internal/manifest"
	"margo/internal/output/html"
	"margo/internal/output/pdf"
	"margo/internal/project"
	"margo/internal/scaffold"
	"margo/internal/serve"
	"margo/internal/theme"
	"margo/internal/version"
)

type commandError struct {
	message string
	report  diagnostics.Report
}

func (e commandError) Error() string {
	return e.message
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		writeHelp(stdout)
		return 0
	}

	err := dispatch(args, stdout, stderr)
	if err == nil {
		return 0
	}

	var cmdErr commandError
	if errors.As(err, &cmdErr) {
		if len(cmdErr.report.Items) > 0 {
			diagnostics.WriteReport(stderr, cmdErr.report)
		}
		if cmdErr.message != "" {
			fmt.Fprintln(stderr, cmdErr.message)
		}
		return 1
	}

	fmt.Fprintln(stderr, err)
	return 1
}

func dispatch(args []string, stdout io.Writer, stderr io.Writer) error {
	switch args[0] {
	case "help", "--help", "-h":
		writeHelp(stdout)
		return nil
	case "version":
		fmt.Fprintf(stdout, "%s %s\n", version.Name, version.Version)
		return nil
	case "build":
		return runBuildLikeCommand("build", args[1:], stdout)
	case "serve":
		return runBuildLikeCommand("serve", args[1:], stdout)
	case "new":
		return runNestedNew(args[1:], stdout, stderr)
	case "init":
		return runInit(stdout)
	case "clean":
		return runClean(stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runNestedNew(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("new requires a deck name or subcommand")
	}

	if args[0] == "slide" {
		return runNewSlide(args[1:], stdout)
	}
	if args[0] == "theme" {
		return runNewTheme(args[1:], stdout)
	}

	switch strings.Join(args[:min(2, len(args))], " ") {
	default:
		targetDir := args[0]
		return runNewDeck(targetDir, stdout)
	}
}

func runNewDeck(target string, stdout io.Writer) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	targetDir := target
	if !filepath.IsAbs(targetDir) {
		targetDir = filepath.Join(wd, target)
	}
	if err := scaffold.CreateDeck(scaffold.DeckOptions{
		Name:      target,
		TargetDir: targetDir,
	}); err != nil {
		return fmt.Errorf("create deck scaffold: %w", err)
	}

	fmt.Fprintf(stdout, "created deck scaffold at %s\n", targetDir)
	fmt.Fprintf(stdout, "next: cd %s && margo build\n", target)
	return nil
}

func runInit(stdout io.Writer) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	name := filepath.Base(wd)
	if err := scaffold.CreateDeck(scaffold.DeckOptions{
		Name:      name,
		TargetDir: wd,
	}); err != nil {
		return fmt.Errorf("initialize deck scaffold: %w", err)
	}

	fmt.Fprintf(stdout, "initialized deck scaffold in %s\n", wd)
	fmt.Fprintln(stdout, "next: margo build")
	return nil
}

func runNewSlide(args []string, stdout io.Writer) error {
	slideName, archetypeName, err := parseNewSlideArgs(args)
	if err != nil {
		return err
	}
	if slideName == "" {
		return fmt.Errorf("new slide requires a slide name")
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	root, err := project.Discover(wd)
	if err != nil {
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "project_not_found",
			Message:  err.Error(),
			Path:     wd,
		})
		return commandError{
			message: "new slide requires a Margo project root",
			report:  report,
		}
	}

	if archetypeName == "" {
		archetypeName, err = chooseSlideArchetype(root.Dir, os.Stdin, stdout)
		if err != nil {
			return err
		}
	}

	indexPath, err := scaffold.CreateSlide(scaffold.SlideOptions{
		ProjectRoot: root.Dir,
		Name:        slideName,
		Archetype:   archetypeName,
	})
	if err != nil {
		return fmt.Errorf("create slide scaffold: %w", err)
	}
	slideID := filepath.Base(filepath.Dir(indexPath))
	if err := manifest.AppendSlide(root.Dir, slideID); err != nil {
		return fmt.Errorf("append new slide to manifest: %w", err)
	}

	fmt.Fprintf(stdout, "created slide at %s\n", indexPath)
	return nil
}

func runNewTheme(args []string, stdout io.Writer) error {
	themeName, blank, err := parseNewThemeArgs(args)
	if err != nil {
		return err
	}
	if themeName == "" {
		return fmt.Errorf("new theme requires a theme name")
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	root, err := project.Discover(wd)
	if err != nil {
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "project_not_found",
			Message:  err.Error(),
			Path:     wd,
		})
		return commandError{
			message: "new theme requires a Margo project root",
			report:  report,
		}
	}

	themeDir, err := scaffold.CreateTheme(scaffold.ThemeOptions{
		ProjectRoot: root.Dir,
		Name:        themeName,
		Blank:       blank,
	})
	if err != nil {
		return fmt.Errorf("create theme scaffold: %w", err)
	}

	fmt.Fprintf(stdout, "created theme at %s\n", themeDir)
	if blank {
		fmt.Fprintln(stdout, "theme mode: blank")
	} else {
		fmt.Fprintln(stdout, "theme mode: default-inspired")
	}
	return nil
}

func runClean(stdout io.Writer) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	root, err := project.Discover(wd)
	if err != nil {
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "project_not_found",
			Message:  err.Error(),
			Path:     wd,
		})
		return commandError{
			message: "clean requires a Margo project root",
			report:  report,
		}
	}

	if err := clean.Project(root.Dir); err != nil {
		return fmt.Errorf("clean project outputs: %w", err)
	}

	fmt.Fprintf(stdout, "cleaned generated output in %s\n", root.Dir)
	return nil
}

func runBuildLikeCommand(name string, args []string, stdout io.Writer) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	includeDrafts, openBrowser, err := parseBuildLikeArgs(name, args)
	if err != nil {
		return err
	}

	root, err := project.Discover(wd)
	if err != nil {
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "project_not_found",
			Message:  err.Error(),
			Path:     wd,
		})
		return commandError{
			message: fmt.Sprintf("%s requires a Margo project root", name),
			report:  report,
		}
	}

	raw, err := config.LoadRaw(root.ConfigPath)
	if err != nil {
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "config_load_failed",
			Message:  err.Error(),
			Path:     root.ConfigPath,
		})
		return commandError{
			message: fmt.Sprintf("%s could not load the root config", name),
			report:  report,
		}
	}

	parsed, err := config.Parse(raw)
	if err != nil {
		message := err.Error()
		line := 0
		path := root.ConfigPath
		if fieldErr, ok := config.AsFieldError(err); ok {
			message = fieldErr.Message
			line = fieldErr.Line
			path = fieldErr.Path
		}
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "config_parse_failed",
			Message:  message,
			Path:     path,
			Line:     line,
		})
		return commandError{
			message: fmt.Sprintf("%s could not parse the root config", name),
			report:  report,
		}
	}

	slides, err := content.DiscoverSlides(root.Dir)
	if err != nil {
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "slide_discovery_failed",
			Message:  err.Error(),
			Path:     root.Dir,
		})
		return commandError{
			message: fmt.Sprintf("%s could not discover slide bundles", name),
			report:  report,
		}
	}

	activeTheme, err := theme.Load(root.Dir, parsed.Config.Theme.Name)
	if err != nil {
		message := err.Error()
		path := root.Dir
		line := 0
		if themeErr, ok := theme.AsError(err); ok {
			message = themeErr.Message
			path = themeErr.Path
			line = themeErr.Line
		}
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "theme_load_failed",
			Message:  message,
			Path:     path,
			Line:     line,
		})
		return commandError{
			message: fmt.Sprintf("%s could not load the active theme", name),
			report:  report,
		}
	}
	parsed.Config.Theme.Options, err = theme.ResolveOptions(activeTheme, parsed.Config.Theme.Options)
	if err != nil {
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "theme_option_validation_failed",
			Message:  err.Error(),
			Path:     root.ConfigPath,
		})
		return commandError{
			message: fmt.Sprintf("%s could not resolve theme options", name),
			report:  report,
		}
	}

	manifestFile, hasManifest, err := manifest.Load(root.Dir)
	if err != nil {
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "manifest_load_failed",
			Message:  err.Error(),
			Path:     root.Dir,
		})
		return commandError{
			message: fmt.Sprintf("%s could not load the deck manifest", name),
			report:  report,
		}
	}

	fmt.Fprintf(stdout, "%s: discovered project root %s\n", name, root.Dir)
	fmt.Fprintf(stdout, "%s: loaded %s (%d bytes)\n", name, raw.Path, len(raw.Bytes))
	fmt.Fprintf(stdout, "%s: discovered %d slide bundles\n", name, len(slides))
	if hasManifest {
		fmt.Fprintf(stdout, "%s: loaded manifest %s\n", name, manifest.Filename)
	}
	fmt.Fprintf(stdout, "%s: loaded theme %s\n", name, activeTheme.Name)

	resolveSlides := func(includeDrafts bool) ([]deck.Slide, error) {
		resolvedSlides, err := content.DiscoverSlides(root.Dir)
		if err != nil {
			return nil, err
		}
		if hasManifest {
			resolvedSlides, err = manifest.Apply(resolvedSlides, manifestFile)
			if err != nil {
				return nil, err
			}
		}
		return deck.FilterSlides(resolvedSlides, deck.FilterOptions{
			IncludeDrafts: includeDrafts,
		}), nil
	}

	rebuild := func() error {
		raw, err := config.LoadRaw(root.ConfigPath)
		if err != nil {
			return err
		}
		parsed, err := config.Parse(raw)
		if err != nil {
			return err
		}
		activeTheme, err := theme.Load(root.Dir, parsed.Config.Theme.Name)
		if err != nil {
			return err
		}
		parsed.Config.Theme.Options, err = theme.ResolveOptions(activeTheme, parsed.Config.Theme.Options)
		if err != nil {
			return err
		}
		slides, err = resolveSlides(includeDrafts)
		if err != nil {
			return err
		}
		slides = deck.ApplySectionDividers(slides)
		model := deck.Model{
			Config:   parsed.Config,
			Sections: deck.BuildSections(slides),
			Slides:   slides,
		}
		renderPDF := name == "build" && parsed.Config.Outputs.PDF
		if parsed.Config.Outputs.HTML || renderPDF {
			report, err := html.Write(root.Dir, model, activeTheme)
			if err != nil {
				return err
			}
			if len(report.Items) > 0 {
				diagnostics.WriteReport(stdout, report)
			}
		}
		if renderPDF {
			if err := pdf.Write(root.Dir); err != nil {
				return err
			}
		}
		return nil
	}

	if name == "build" && (parsed.Config.Outputs.HTML || parsed.Config.Outputs.PDF) {
		if err := rebuild(); err != nil {
			return fmt.Errorf("build outputs: %w", err)
		}
		filteredSlides, err := resolveSlides(includeDrafts)
		if err != nil {
			return fmt.Errorf("resolve filtered slides: %w", err)
		}
		filteredSlides = deck.ApplySectionDividers(filteredSlides)
		fmt.Fprintf(stdout, "%s: rendering %d slides after filtering\n", name, len(filteredSlides))
		if parsed.Config.Outputs.HTML || parsed.Config.Outputs.PDF {
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, html.OutputFile)
		}
		if parsed.Config.Outputs.PDF {
			if browser, browserErr := pdf.DetectBrowser(); browserErr == nil {
				fmt.Fprintf(stdout, "%s: pdf browser %s (%s)\n", name, browser.Path, browser.Source)
			}
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, pdf.OutputFile)
		}
		return nil
	}

	if name == "serve" {
		if parsed.Config.Outputs.HTML {
			if err := rebuild(); err != nil {
				return fmt.Errorf("write html output: %w", err)
			}
			filteredSlides, err := resolveSlides(includeDrafts)
			if err != nil {
				return fmt.Errorf("resolve filtered slides: %w", err)
			}
			filteredSlides = deck.ApplySectionDividers(filteredSlides)
			fmt.Fprintf(stdout, "%s: rendering %d slides after filtering\n", name, len(filteredSlides))
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, html.OutputFile)
		}

		generatePDF := func() error {
			if !parsed.Config.Outputs.PDF {
				return fmt.Errorf("pdf output is not enabled")
			}
			if err := rebuild(); err != nil {
				return err
			}
			return pdf.Write(root.Dir)
		}
		return serve.Start(root.Dir, rebuild, serve.Options{
			OpenBrowser: openBrowser,
			PDFEnabled:  parsed.Config.Outputs.PDF,
			GeneratePDF: generatePDF,
		})
	}

	fmt.Fprintf(stdout, "%s: not implemented\n", name)
	return nil
}

func notImplemented(name string) error {
	return fmt.Errorf("%s: not implemented", name)
}

func writeHelp(w io.Writer) {
	fmt.Fprintf(w, "%s %s\n\n", version.Name, version.Version)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  margo <command> [arguments]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  build        Build all configured outputs for the current deck")
	fmt.Fprintln(w, "  serve        Serve the current deck locally")
	fmt.Fprintln(w, "  new          Create a deck, slide, or theme scaffold")
	fmt.Fprintln(w, "  init         Initialize a deck in the current directory")
	fmt.Fprintln(w, "  clean        Remove generated output and tool-managed build state")
	fmt.Fprintln(w, "  version      Print version information")
}

func parseBuildLikeArgs(command string, args []string) (bool, bool, error) {
	includeDrafts := command == "serve"
	openBrowser := command == "serve"

	for _, arg := range args {
		switch arg {
		case "--include-drafts":
			includeDrafts = true
		case "--no-open":
			if command != "serve" {
				return false, false, fmt.Errorf("%s does not support %s", command, arg)
			}
			openBrowser = false
		default:
			return false, false, fmt.Errorf("unknown %s option %q", command, arg)
		}
	}

	return includeDrafts, openBrowser, nil
}

func parseNewSlideArgs(args []string) (string, string, error) {
	var slideName string
	var archetypeName string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--archetype":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("new slide requires a value for --archetype")
			}
			archetypeName = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return "", "", fmt.Errorf("unknown new slide option %q", args[i])
			}
			if slideName != "" {
				return "", "", fmt.Errorf("new slide accepts exactly one slide name")
			}
			slideName = args[i]
		}
	}

	return slideName, archetypeName, nil
}

func chooseSlideArchetype(projectRoot string, stdin *os.File, stdout io.Writer) (string, error) {
	return chooseSlideArchetypeFromReader(projectRoot, stdin, stdout, isInteractiveStdin(stdin))
}

func chooseSlideArchetypeFromReader(projectRoot string, input io.Reader, stdout io.Writer, interactive bool) (string, error) {
	available, err := archetype.List(projectRoot)
	if err != nil {
		return "", fmt.Errorf("list archetypes: %w", err)
	}
	if len(available) == 0 {
		return "default", nil
	}
	if len(available) == 1 {
		return available[0].Name, nil
	}
	if !interactive {
		return available[0].Name, nil
	}

	fmt.Fprintln(stdout, "choose an archetype for the new slide:")
	for i, meta := range available {
		label := meta.Name
		if strings.TrimSpace(meta.Description) != "" {
			label += " - " + meta.Description
		}
		fmt.Fprintf(stdout, "  %d. %s\n", i+1, label)
	}

	reader := bufio.NewReader(input)
	for {
		fmt.Fprintf(stdout, "select archetype [1]: ")
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read archetype selection: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return available[0].Name, nil
		}
		choice, convErr := strconv.Atoi(line)
		if convErr == nil && choice >= 1 && choice <= len(available) {
			return available[choice-1].Name, nil
		}
		fmt.Fprintln(stdout, "invalid selection; enter a number from the list")
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("invalid archetype selection %q", line)
		}
	}
}

func isInteractiveStdin(stdin *os.File) bool {
	if stdin == nil {
		return false
	}
	info, err := stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func parseNewThemeArgs(args []string) (string, bool, error) {
	var themeName string
	blank := false

	for _, arg := range args {
		switch arg {
		case "--blank", "blank":
			blank = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", false, fmt.Errorf("unknown new theme option %q", arg)
			}
			if themeName != "" {
				return "", false, fmt.Errorf("new theme accepts exactly one theme name")
			}
			themeName = arg
		}
	}

	return themeName, blank, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
