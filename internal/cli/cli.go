package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"margo/internal/clean"
	"margo/internal/config"
	"margo/internal/content"
	"margo/internal/deck"
	"margo/internal/diagnostics"
	"margo/internal/manifest"
	"margo/internal/output/html"
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
		return runBuildLikeCommand("build", stdout)
	case "serve":
		return runBuildLikeCommand("serve", stdout)
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

	targetDir := filepath.Join(wd, target)
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
	if len(args) == 0 {
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

	indexPath, err := scaffold.CreateSlide(scaffold.SlideOptions{
		ProjectRoot: root.Dir,
		Name:        args[0],
		Archetype:   "default",
	})
	if err != nil {
		return fmt.Errorf("create slide scaffold: %w", err)
	}

	fmt.Fprintf(stdout, "created slide at %s\n", indexPath)
	return nil
}

func runNewTheme(args []string, stdout io.Writer) error {
	if len(args) == 0 {
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

	name := args[0]
	blank := len(args) > 1 && args[1] == "blank"
	themeDir, err := scaffold.CreateTheme(scaffold.ThemeOptions{
		ProjectRoot: root.Dir,
		Name:        name,
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

func runBuildLikeCommand(name string, stdout io.Writer) error {
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
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "config_parse_failed",
			Message:  err.Error(),
			Path:     root.ConfigPath,
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
		report := diagnostics.Report{}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Code:     "theme_load_failed",
			Message:  err.Error(),
			Path:     root.Dir,
		})
		return commandError{
			message: fmt.Sprintf("%s could not load the active theme", name),
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

	rebuild := func() error {
		raw, err := config.LoadRaw(root.ConfigPath)
		if err != nil {
			return err
		}
		parsed, err := config.Parse(raw)
		if err != nil {
			return err
		}
		slides, err := content.DiscoverSlides(root.Dir)
		if err != nil {
			return err
		}
		activeTheme, err := theme.Load(root.Dir, parsed.Config.Theme.Name)
		if err != nil {
			return err
		}
		manifestFile, hasManifest, err := manifest.Load(root.Dir)
		if err != nil {
			return err
		}
		if hasManifest {
			slides, err = manifest.Apply(slides, manifestFile)
			if err != nil {
				return err
			}
		}
		includeDrafts := name == "serve"
		slides = deck.FilterSlides(slides, deck.FilterOptions{
			IncludeDrafts: includeDrafts,
		})
		model := deck.Model{
			Config:   parsed.Config,
			Sections: deck.BuildSections(slides),
			Slides:   slides,
		}
		if !parsed.Config.Outputs.HTML {
			return nil
		}
		return html.Write(root.Dir, model, activeTheme)
	}

	if name == "build" && parsed.Config.Outputs.HTML {
		if err := rebuild(); err != nil {
			return fmt.Errorf("write html output: %w", err)
		}
		if hasManifest {
			slides, err = manifest.Apply(slides, manifestFile)
			if err != nil {
				return fmt.Errorf("apply manifest: %w", err)
			}
		}
		filteredSlides := deck.FilterSlides(slides, deck.FilterOptions{})
		fmt.Fprintf(stdout, "%s: rendering %d slides after filtering\n", name, len(filteredSlides))
		fmt.Fprintf(stdout, "%s: wrote %s\n", name, html.OutputFile)
		return nil
	}

	if name == "serve" {
		if parsed.Config.Outputs.HTML {
			if err := rebuild(); err != nil {
				return fmt.Errorf("write html output: %w", err)
			}
			if hasManifest {
				slides, err = manifest.Apply(slides, manifestFile)
				if err != nil {
					return fmt.Errorf("apply manifest: %w", err)
				}
			}
			filteredSlides := deck.FilterSlides(slides, deck.FilterOptions{
				IncludeDrafts: true,
			})
			fmt.Fprintf(stdout, "%s: rendering %d slides after filtering\n", name, len(filteredSlides))
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, html.OutputFile)
		}
		return serve.Start(root.Dir, rebuild)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
