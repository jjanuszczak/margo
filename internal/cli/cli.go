package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jjanuszczak/margo/internal/archetype"
	"github.com/jjanuszczak/margo/internal/clean"
	"github.com/jjanuszczak/margo/internal/config"
	"github.com/jjanuszczak/margo/internal/content"
	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/deploy"
	"github.com/jjanuszczak/margo/internal/diagnostics"
	"github.com/jjanuszczak/margo/internal/manifest"
	"github.com/jjanuszczak/margo/internal/output/html"
	"github.com/jjanuszczak/margo/internal/output/pdf"
	"github.com/jjanuszczak/margo/internal/output/png"
	"github.com/jjanuszczak/margo/internal/output/pptx"
	"github.com/jjanuszczak/margo/internal/output/printhtml"
	"github.com/jjanuszczak/margo/internal/project"
	"github.com/jjanuszczak/margo/internal/projectarchive"
	"github.com/jjanuszczak/margo/internal/scaffold"
	"github.com/jjanuszczak/margo/internal/serve"
	"github.com/jjanuszczak/margo/internal/skillinstall"
	"github.com/jjanuszczak/margo/internal/theme"
	"github.com/jjanuszczak/margo/internal/themearchive"
	"github.com/jjanuszczak/margo/internal/upgrade"
	"github.com/jjanuszczak/margo/internal/version"
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
		fmt.Fprintf(stdout, "%s %s\n", version.Name, version.Current())
		return nil
	case "build":
		return runBuildLikeCommand("build", args[1:], stdout)
	case "serve":
		return runBuildLikeCommand("serve", args[1:], stdout)
	case "pack":
		return runPack(args[1:], stdout)
	case "unpack":
		return runUnpack(args[1:], stdout)
	case "new":
		return runNestedNew(args[1:], stdout, stderr)
	case "slide":
		return runSlideCommand(args[1:], stdout)
	case "theme":
		return runThemeCommand(args[1:], stdout)
	case "init":
		return runInit(stdout)
	case "clean":
		return runClean(stdout)
	case "upgrade":
		return runUpgrade(args[1:], stdout)
	case "skills":
		return runSkills(args[1:], stdout)
	case "deploy":
		return runDeploy(args[1:], stdout)
	default:
		if strings.HasSuffix(strings.ToLower(args[0]), projectarchive.Extension) {
			return runArchiveOpen(args, stdout)
		}
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runSkills(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "install" || len(args) < 2 || args[1] != "brand-theme" {
		return errors.New("usage: margo skills install brand-theme --scope user|project [--plan]")
	}
	scope := skillinstall.Scope("")
	planOnly := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--scope":
			if i+1 >= len(args) {
				return errors.New("skills install requires a value for --scope")
			}
			scope = skillinstall.Scope(args[i+1])
			i++
		case "--plan", "--dry-run":
			planOnly = true
		default:
			return fmt.Errorf("unknown skills install option %q", args[i])
		}
	}
	if scope != skillinstall.User && scope != skillinstall.Project {
		return errors.New("skills install requires --scope user or --scope project")
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	projectRoot := ""
	if scope == skillinstall.Project {
		root, err := project.Discover(wd)
		if err != nil {
			return fmt.Errorf("project skill install requires a Margo project root: %w", err)
		}
		projectRoot = root.Dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find user home directory: %w", err)
	}
	target, err := skillinstall.Target(projectRoot, home, scope)
	if err != nil {
		return err
	}
	files, err := skillinstall.Files(scope)
	if err != nil {
		return err
	}
	plan, err := skillinstall.BuildPlan(target, files)
	if err != nil {
		return fmt.Errorf("plan skill install: %w", err)
	}
	fmt.Fprint(stdout, skillinstall.Format(plan))
	if planOnly {
		return nil
	}
	if err := skillinstall.Apply(plan, files); err != nil {
		return fmt.Errorf("install skill: %w", err)
	}
	fmt.Fprintf(stdout, "installed brand theme skill at %s\n", target)
	return nil
}

func runUpgrade(args []string, stdout io.Writer) error {
	apply := false
	for _, arg := range args {
		switch arg {
		case "--plan", "--dry-run":
		case "--apply":
			apply = true
		default:
			return fmt.Errorf("unknown upgrade option %q", arg)
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := project.Discover(wd)
	if err != nil {
		return fmt.Errorf("upgrade requires a Margo project root: %w", err)
	}
	plan, err := upgrade.BuildPlan(root.Dir)
	if err != nil {
		return fmt.Errorf("plan project upgrade: %w", err)
	}
	fmt.Fprint(stdout, upgrade.Format(plan))
	if !apply {
		return nil
	}
	backup, err := upgrade.Apply(root.Dir, plan)
	if err != nil {
		return fmt.Errorf("apply project upgrade: %w", err)
	}
	if backup != "" {
		fmt.Fprintf(stdout, "upgrade applied; backups at %s\n", backup)
	} else {
		fmt.Fprintln(stdout, "upgrade applied; no files needed changes")
	}
	return nil
}

func runDeploy(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "github-pages" {
		return errors.New("usage: margo deploy github-pages [--workflow-name <name>] [--margo-version <version>] [--replace]")
	}
	options := deploy.PagesOptions{}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--workflow-name":
			if i+1 >= len(args) {
				return errors.New("deploy github-pages requires a value for --workflow-name")
			}
			options.WorkflowName = args[i+1]
			i++
		case "--margo-version":
			if i+1 >= len(args) {
				return errors.New("deploy github-pages requires a value for --margo-version")
			}
			options.MargoVersion = args[i+1]
			i++
		case "--replace":
			options.Replace = true
		default:
			return fmt.Errorf("unknown deploy github-pages option %q", args[i])
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := project.Discover(wd)
	if err != nil {
		return fmt.Errorf("deploy github-pages requires a Margo project root: %w", err)
	}
	if options.MargoVersion == "" {
		options.MargoVersion = version.Current()
	}
	if options.MargoVersion == "0.0.0-dev" {
		return errors.New("deploy github-pages requires --margo-version when running an unversioned development build")
	}
	result, err := deploy.GitHubPages(root.Dir, options)
	if err != nil {
		return fmt.Errorf("configure GitHub Pages deployment: %w", err)
	}
	fmt.Fprintf(stdout, "created GitHub Pages workflow at %s\n", result.WorkflowPath)
	fmt.Fprintln(stdout, "next: commit these files, push a v* tag or run the workflow manually, and set Pages source to GitHub Actions in repository settings")
	return nil
}

func runPack(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("pack requires exactly one deck directory")
	}
	root, err := project.Discover(args[0])
	if err != nil {
		return fmt.Errorf("pack requires a Margo project directory: %w", err)
	}
	outputPath := filepath.Join(filepath.Dir(root.Dir), filepath.Base(root.Dir)+projectarchive.Extension)
	if err := projectarchive.Pack(root.Dir, outputPath, version.Current()); err != nil {
		return fmt.Errorf("pack project archive: %w", err)
	}
	fmt.Fprintf(stdout, "packed project archive at %s\n", outputPath)
	return nil
}

func runUnpack(args []string, stdout io.Writer) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("unpack requires an archive and optional destination")
	}
	archivePath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolve archive path: %w", err)
	}
	destination := ""
	if len(args) == 2 {
		destination = args[1]
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		destination = filepath.Join(wd, strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath)))
	}
	if !filepath.IsAbs(destination) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		destination = filepath.Join(wd, destination)
	}
	manifest, err := projectarchive.Unpack(archivePath, destination)
	if err != nil {
		return fmt.Errorf("unpack project archive: %w", err)
	}
	fmt.Fprintf(stdout, "unpacked %s project at %s\n", manifest.ProjectName, destination)
	return nil
}

func runArchiveOpen(args []string, stdout io.Writer) error {
	archivePath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolve archive path: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "margo-archive-")
	if err != nil {
		return fmt.Errorf("create temporary archive workspace: %w", err)
	}
	defer os.RemoveAll(tempDir)
	if _, err := projectarchive.Unpack(archivePath, tempDir); err != nil {
		return fmt.Errorf("open project archive: %w", err)
	}
	previousDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		return fmt.Errorf("enter temporary archive workspace: %w", err)
	}
	defer os.Chdir(previousDir)
	fmt.Fprintf(stdout, "opened archive in temporary workspace %s\n", tempDir)
	return runBuildLikeCommand("serve", args[1:], stdout)
}

func runNestedNew(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("new requires a deck name or subcommand")
	}

	if args[0] == "slide" {
		return runNewSlide(args[1:], stdout)
	}
	if args[0] == "note" {
		return runNewNote(args[1:], stdout)
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

func runThemeCommand(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("theme requires a subcommand")
	}
	switch args[0] {
	case "add":
		return runThemeAdd(args[1:], stdout)
	case "pack":
		return runThemePack(args[1:], stdout)
	case "import":
		return runThemeImport(args[1:], stdout)
	case "update":
		return runThemeUpdate(args[1:], stdout)
	case "list":
		return runThemeList(stdout)
	case "pptx":
		return runThemePPTX(args[1:], stdout)
	default:
		return fmt.Errorf("unknown theme subcommand %q", args[0])
	}
}

func runThemePPTX(args []string, stdout io.Writer) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("theme pptx requires init, inspect, or validate and a theme name")
	}
	action, themeName := args[0], ""
	if len(args) == 2 {
		themeName = args[1]
	}
	if themeName == "" {
		return fmt.Errorf("theme pptx %s requires a theme name", action)
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	root, err := project.Discover(wd)
	if err != nil {
		return fmt.Errorf("theme pptx %s requires a Margo project root: %w", action, err)
	}
	active, err := theme.Load(root.Dir, themeName)
	if err != nil {
		return err
	}
	switch action {
	case "validate":
		fmt.Fprintf(stdout, "validated PPTX contract for theme %s\n", active.Name)
		if active.PPTX == nil {
			fmt.Fprintln(stdout, "PPTX contract: generic fallback")
		}
		return nil
	case "inspect":
		fmt.Fprintf(stdout, "theme: %s\n", active.Name)
		fmt.Fprintf(stdout, "html layouts: %d\n", len(active.SlideLayouts))
		if active.PPTX == nil {
			fmt.Fprintln(stdout, "PPTX contract: not configured (generic fallback will be used)")
			inspectThemeCandidates(active, stdout)
			return nil
		}
		fmt.Fprintf(stdout, "PPTX slide size: %s\n", active.PPTX.SlideSize)
		fmt.Fprintf(stdout, "PPTX layouts: %d\n", len(active.PPTX.Layouts))
		fmt.Fprintf(stdout, "PPTX assets: %d\n", len(active.PPTX.Assets))
		return nil
	case "init":
		return initThemePPTX(active, stdout)
	default:
		return fmt.Errorf("unknown theme pptx action %q", action)
	}
}

func inspectThemeCandidates(active theme.Metadata, stdout io.Writer) {
	cssPath := filepath.Join(active.RootDir, "assets", "theme.css")
	css, err := os.ReadFile(cssPath)
	if err == nil {
		fonts := regexp.MustCompile(`(?i)font-family\s*:\s*([^;}{]+)`).FindStringSubmatch(string(css))
		if len(fonts) == 2 {
			fmt.Fprintf(stdout, "candidate body font: %s\n", strings.TrimSpace(fonts[1]))
		}
		colors := regexp.MustCompile(`#(?:[0-9a-fA-F]{6})`).FindAllString(string(css), 6)
		if len(colors) > 0 {
			fmt.Fprintf(stdout, "candidate CSS colors: %s\n", strings.Join(colors, ", "))
		}
	}
	if entries, err := os.ReadDir(filepath.Join(active.RootDir, "assets")); err == nil {
		var assets []string
		for _, entry := range entries {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if !entry.IsDir() && (ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".svg") {
				assets = append(assets, entry.Name())
			}
		}
		if len(assets) > 0 {
			fmt.Fprintf(stdout, "candidate image assets: %s\n", strings.Join(assets, ", "))
		}
	}
}

func initThemePPTX(active theme.Metadata, stdout io.Writer) error {
	if active.PPTX != nil {
		fmt.Fprintf(stdout, "PPTX theme contract already configured in %s\n", filepath.Join(active.RootDir, theme.ThemeMetadataFile))
		return nil
	}
	contractDir := filepath.Join(active.RootDir, "pptx")
	contractPath := filepath.Join(contractDir, "theme.yaml")
	if _, err := os.Stat(contractPath); err == nil {
		return fmt.Errorf("PPTX theme contract already exists: %s", contractPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat PPTX theme contract: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(contractDir, "layouts"), 0o755); err != nil {
		return fmt.Errorf("create PPTX theme directory: %w", err)
	}
	contract := "slide_size: widescreen\nfonts:\n  heading: Aptos Display\n  body: Aptos\ncolors:\n  background: \"#FFFFFF\"\n  foreground: \"#1F2937\"\n  accent: \"#8F6F33\"\nassets: {}\nlayouts: {}\n"
	if err := os.WriteFile(contractPath, []byte(contract), 0o644); err != nil {
		return fmt.Errorf("write PPTX theme contract: %w", err)
	}
	fmt.Fprintf(stdout, "created PPTX theme contract at %s\n", contractPath)
	return nil
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

type insertSlideOptions struct {
	Name      string
	Archetype string
	Before    string
	After     string
	Position  int
	Renumber  bool
}

func runSlideCommand(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: margo slide <insert|move|delete> ...")
	}
	switch args[0] {
	case "insert":
		return runSlideInsert(args[1:], stdout)
	case "move":
		return runSlideMove(args[1:], stdout)
	case "delete":
		return runSlideDelete(args[1:], stdout)
	default:
		return fmt.Errorf("unknown slide command %q", args[0])
	}
}

func runSlideInsert(args []string, stdout io.Writer) error {
	opts, err := parseInsertSlideArgs(args)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	root, err := project.Discover(wd)
	if err != nil {
		return fmt.Errorf("slide insert requires a Margo project root: %w", err)
	}

	slides, err := content.DiscoverSlides(root.Dir)
	if err != nil {
		return fmt.Errorf("discover slides: %w", err)
	}
	manifestFile, hasManifest, err := manifest.Load(root.Dir)
	if err != nil {
		return err
	}
	if hasManifest {
		slides, err = manifest.Apply(slides, manifestFile)
		if err != nil {
			return fmt.Errorf("apply manifest: %w", err)
		}
	}

	insertAt, err := resolveInsertPosition(slides, opts)
	if err != nil {
		return err
	}
	if opts.Archetype == "" {
		opts.Archetype, err = chooseSlideArchetype(root.Dir, os.Stdin, stdout)
		if err != nil {
			return err
		}
	}
	indexPath, err := scaffold.CreateSlide(scaffold.SlideOptions{ProjectRoot: root.Dir, Name: opts.Name, Archetype: opts.Archetype})
	if err != nil {
		return fmt.Errorf("create slide scaffold: %w", err)
	}
	newID := filepath.Base(filepath.Dir(indexPath))
	orderedIDs := make([]string, 0, len(slides)+1)
	for i, slide := range slides {
		if i == insertAt {
			orderedIDs = append(orderedIDs, newID)
		}
		orderedIDs = append(orderedIDs, slide.ID)
	}
	if insertAt == len(slides) {
		orderedIDs = append(orderedIDs, newID)
	}

	shouldRenumber := opts.Renumber || hasPositionalBundleNames(slides)
	finalIDs := append([]string(nil), orderedIDs...)
	if shouldRenumber {
		finalIDs = renumberBundleIDs(orderedIDs)
	}
	if err := renameSlideBundles(root.Dir, orderedIDs, finalIDs); err != nil {
		return fmt.Errorf("rename slide bundles: %w", err)
	}
	if err := rewriteSlideOrders(root.Dir, finalIDs); err != nil {
		return fmt.Errorf("update slide order: %w", err)
	}
	if hasManifest {
		if err := manifest.Save(root.Dir, manifest.File{Slides: finalIDs}); err != nil {
			return fmt.Errorf("save manifest: %w", err)
		}
	}

	fmt.Fprintf(stdout, "inserted %s at position %d\n", finalIDs[insertAt], insertAt+1)
	for i := range orderedIDs {
		if orderedIDs[i] == newID {
			fmt.Fprintf(stdout, "created %s\n", finalIDs[i])
		} else if orderedIDs[i] != finalIDs[i] {
			fmt.Fprintf(stdout, "renamed %s -> %s\n", orderedIDs[i], finalIDs[i])
		}
	}
	return nil
}

func parseInsertSlideArgs(args []string) (insertSlideOptions, error) {
	var opts insertSlideOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--archetype", "--before", "--after", "--position":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("slide insert requires a value for %s", args[i])
			}
			value := strings.TrimSpace(args[i+1])
			switch args[i] {
			case "--archetype":
				if opts.Archetype != "" {
					return opts, errors.New("slide insert accepts --archetype once")
				}
				opts.Archetype = value
			case "--before":
				if opts.Before != "" {
					return opts, errors.New("slide insert accepts --before once")
				}
				opts.Before = value
			case "--after":
				if opts.After != "" {
					return opts, errors.New("slide insert accepts --after once")
				}
				opts.After = value
			case "--position":
				if opts.Position != 0 {
					return opts, errors.New("slide insert accepts --position once")
				}
				position, err := strconv.Atoi(value)
				if err != nil || position < 1 {
					return opts, fmt.Errorf("slide insert position %q must be a positive integer", value)
				}
				opts.Position = position
			}
			i++
		case "--renumber":
			opts.Renumber = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return opts, fmt.Errorf("unknown slide insert option %q", args[i])
			}
			if opts.Name != "" {
				return opts, errors.New("slide insert accepts exactly one slide name")
			}
			opts.Name = args[i]
		}
	}
	if opts.Name == "" {
		return opts, errors.New("slide insert requires a slide name")
	}
	selectors := 0
	if opts.Before != "" {
		selectors++
	}
	if opts.After != "" {
		selectors++
	}
	if opts.Position != 0 {
		selectors++
	}
	if selectors != 1 {
		return opts, errors.New("slide insert requires exactly one of --before, --after, or --position")
	}
	return opts, nil
}

func resolveInsertPosition(slides []deck.Slide, opts insertSlideOptions) (int, error) {
	if opts.Position != 0 {
		if opts.Position > len(slides)+1 {
			return 0, fmt.Errorf("slide insert position %d is outside this %d-slide deck", opts.Position, len(slides))
		}
		return opts.Position - 1, nil
	}
	for i, slide := range slides {
		if slide.ID == opts.Before {
			return i, nil
		}
		if slide.ID == opts.After {
			return i + 1, nil
		}
	}
	target := opts.Before
	if target == "" {
		target = opts.After
	}
	return 0, fmt.Errorf("slide insert target %q was not found", target)
}

type sequenceOptions struct {
	Before   string
	After    string
	Position int
	Renumber bool
}

func runSlideMove(args []string, stdout io.Writer) error {
	slideID, opts, err := parseSlideMoveArgs(args)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	root, err := project.Discover(wd)
	if err != nil {
		return fmt.Errorf("slide move requires a Margo project root: %w", err)
	}
	slides, hasManifest, err := loadResolvedSlides(root.Dir)
	if err != nil {
		return err
	}
	if opts.Before == slideID || opts.After == slideID {
		return fmt.Errorf("slide move cannot place %q relative to itself", slideID)
	}
	moveAt := -1
	remaining := make([]deck.Slide, 0, len(slides)-1)
	for i, slide := range slides {
		if slide.ID == slideID {
			moveAt = i
			continue
		}
		remaining = append(remaining, slide)
	}
	if moveAt < 0 {
		return fmt.Errorf("slide move target %q was not found", slideID)
	}
	insertAt, err := resolveSequencePosition(remaining, opts, "move")
	if err != nil {
		return err
	}
	orderedIDs := make([]string, 0, len(slides))
	for i, slide := range remaining {
		if i == insertAt {
			orderedIDs = append(orderedIDs, slideID)
		}
		orderedIDs = append(orderedIDs, slide.ID)
	}
	if insertAt == len(remaining) {
		orderedIDs = append(orderedIDs, slideID)
	}
	finalIDs := finalSlideIDs(orderedIDs, opts.Renumber || hasPositionalBundleNames(slides))
	if err := applySlideSequence(root.Dir, orderedIDs, finalIDs, hasManifest); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "moved %s to position %d\n", finalIDs[insertAt], insertAt+1)
	writeSlideRenameMap(stdout, orderedIDs, finalIDs, "")
	return nil
}

func runSlideDelete(args []string, stdout io.Writer) error {
	slideID, renumber, err := parseSlideDeleteArgs(args)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	root, err := project.Discover(wd)
	if err != nil {
		return fmt.Errorf("slide delete requires a Margo project root: %w", err)
	}
	slides, hasManifest, err := loadResolvedSlides(root.Dir)
	if err != nil {
		return err
	}
	if len(slides) <= 1 {
		return errors.New("slide delete cannot leave a deck without slides")
	}
	remaining := make([]deck.Slide, 0, len(slides)-1)
	found := false
	for _, slide := range slides {
		if slide.ID == slideID {
			found = true
			continue
		}
		remaining = append(remaining, slide)
	}
	if !found {
		return fmt.Errorf("slide delete target %q was not found", slideID)
	}
	trashPath, err := moveSlideToTrash(root.Dir, slideID)
	if err != nil {
		return fmt.Errorf("move slide to trash: %w", err)
	}
	orderedIDs := make([]string, len(remaining))
	for i, slide := range remaining {
		orderedIDs[i] = slide.ID
	}
	finalIDs := finalSlideIDs(orderedIDs, renumber || hasPositionalBundleNames(slides))
	if err := applySlideSequence(root.Dir, orderedIDs, finalIDs, hasManifest); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "deleted %s (moved to %s)\n", slideID, trashPath)
	writeSlideRenameMap(stdout, orderedIDs, finalIDs, "")
	return nil
}

func parseSlideMoveArgs(args []string) (string, sequenceOptions, error) {
	var slideID string
	opts, err := parseSequenceOptions(args, true)
	if err != nil {
		return "", opts, err
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "--before" || args[i] == "--after" || args[i] == "--position" {
			i++
			continue
		}
		if !strings.HasPrefix(args[i], "--") && slideID == "" {
			slideID = args[i]
		}
	}
	if slideID == "" {
		return "", opts, errors.New("slide move requires a slide bundle name")
	}
	return slideID, opts, nil
}

func parseSlideDeleteArgs(args []string) (string, bool, error) {
	var slideID string
	renumber := false
	for _, arg := range args {
		switch arg {
		case "--renumber":
			renumber = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", false, fmt.Errorf("unknown slide delete option %q", arg)
			}
			if slideID != "" {
				return "", false, errors.New("slide delete accepts exactly one slide bundle name")
			}
			slideID = arg
		}
	}
	if slideID == "" {
		return "", false, errors.New("slide delete requires a slide bundle name")
	}
	return slideID, renumber, nil
}

func parseSequenceOptions(args []string, requirePosition bool) (sequenceOptions, error) {
	var opts sequenceOptions
	var slideIDSeen bool
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--before", "--after", "--position":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("slide move requires a value for %s", args[i])
			}
			value := strings.TrimSpace(args[i+1])
			switch args[i] {
			case "--before":
				if opts.Before != "" {
					return opts, errors.New("slide move accepts --before once")
				}
				opts.Before = value
			case "--after":
				if opts.After != "" {
					return opts, errors.New("slide move accepts --after once")
				}
				opts.After = value
			case "--position":
				if opts.Position != 0 {
					return opts, errors.New("slide move accepts --position once")
				}
				position, err := strconv.Atoi(value)
				if err != nil || position < 1 {
					return opts, fmt.Errorf("slide move position %q must be a positive integer", value)
				}
				opts.Position = position
			}
			i++
		case "--renumber":
			opts.Renumber = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return opts, fmt.Errorf("unknown slide move option %q", args[i])
			}
			if slideIDSeen {
				return opts, errors.New("slide move accepts exactly one slide bundle name")
			}
			slideIDSeen = true
		}
	}
	selectors := 0
	if opts.Before != "" {
		selectors++
	}
	if opts.After != "" {
		selectors++
	}
	if opts.Position != 0 {
		selectors++
	}
	if requirePosition && selectors != 1 {
		return opts, errors.New("slide move requires exactly one of --before, --after, or --position")
	}
	return opts, nil
}

func loadResolvedSlides(projectRoot string) ([]deck.Slide, bool, error) {
	slides, err := content.DiscoverSlides(projectRoot)
	if err != nil {
		return nil, false, fmt.Errorf("discover slides: %w", err)
	}
	manifestFile, hasManifest, err := manifest.Load(projectRoot)
	if err != nil {
		return nil, false, err
	}
	if hasManifest {
		slides, err = manifest.Apply(slides, manifestFile)
		if err != nil {
			return nil, false, fmt.Errorf("apply manifest: %w", err)
		}
	}
	return slides, hasManifest, nil
}

func resolveSequencePosition(slides []deck.Slide, opts sequenceOptions, command string) (int, error) {
	if opts.Position != 0 {
		if opts.Position > len(slides)+1 {
			return 0, fmt.Errorf("slide %s position %d is outside this %d-slide deck", command, opts.Position, len(slides))
		}
		return opts.Position - 1, nil
	}
	for i, slide := range slides {
		if slide.ID == opts.Before {
			return i, nil
		}
		if slide.ID == opts.After {
			return i + 1, nil
		}
	}
	target := opts.Before
	if target == "" {
		target = opts.After
	}
	return 0, fmt.Errorf("slide %s target %q was not found", command, target)
}

func finalSlideIDs(orderedIDs []string, renumber bool) []string {
	if renumber {
		return renumberBundleIDs(orderedIDs)
	}
	return append([]string(nil), orderedIDs...)
}

func applySlideSequence(projectRoot string, orderedIDs, finalIDs []string, hasManifest bool) error {
	if err := renameSlideBundles(projectRoot, orderedIDs, finalIDs); err != nil {
		return fmt.Errorf("rename slide bundles: %w", err)
	}
	if err := rewriteSlideOrders(projectRoot, finalIDs); err != nil {
		return fmt.Errorf("update slide order: %w", err)
	}
	if hasManifest {
		if err := manifest.Save(projectRoot, manifest.File{Slides: finalIDs}); err != nil {
			return fmt.Errorf("save manifest: %w", err)
		}
	}
	return nil
}

func moveSlideToTrash(projectRoot, slideID string) (string, error) {
	trashDir := filepath.Join(projectRoot, ".margo-trash", time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return "", err
	}
	source := filepath.Join(projectRoot, "slides", slideID)
	destination := filepath.Join(trashDir, slideID)
	if err := os.Rename(source, destination); err != nil {
		return "", err
	}
	return destination, nil
}

func writeSlideRenameMap(stdout io.Writer, orderedIDs, finalIDs []string, createdID string) {
	for i := range orderedIDs {
		if orderedIDs[i] == createdID {
			fmt.Fprintf(stdout, "created %s\n", finalIDs[i])
		} else if orderedIDs[i] != finalIDs[i] {
			fmt.Fprintf(stdout, "renamed %s -> %s\n", orderedIDs[i], finalIDs[i])
		}
	}
}

var positionalBundleName = regexp.MustCompile(`^\d{2,}-.+$`)
var positionalBundlePrefix = regexp.MustCompile(`^\d{2,}-(.+)$`)

func hasPositionalBundleNames(slides []deck.Slide) bool {
	return len(slides) > 0 && func() bool {
		for _, slide := range slides {
			if !positionalBundleName.MatchString(slide.ID) {
				return false
			}
		}
		return true
	}()
}

func renumberBundleIDs(ids []string) []string {
	width := len(strconv.Itoa(len(ids)))
	if width < 2 {
		width = 2
	}
	result := make([]string, len(ids))
	for i, id := range ids {
		slug := id
		if matches := positionalBundlePrefix.FindStringSubmatch(id); len(matches) == 2 {
			slug = matches[1]
		}
		result[i] = fmt.Sprintf("%0*d-%s", width, i+1, slug)
	}
	return result
}

func renameSlideBundles(projectRoot string, oldIDs, newIDs []string) error {
	if len(oldIDs) != len(newIDs) {
		return errors.New("slide rename plan is inconsistent")
	}
	seen := make(map[string]bool, len(newIDs))
	for _, id := range newIDs {
		if seen[id] {
			return fmt.Errorf("slide rename plan creates duplicate bundle %q", id)
		}
		seen[id] = true
	}
	type rename struct{ oldPath, tempPath, newPath string }
	var plan []rename
	for i := range oldIDs {
		if oldIDs[i] == newIDs[i] {
			continue
		}
		oldPath := filepath.Join(projectRoot, "slides", oldIDs[i])
		plan = append(plan, rename{oldPath, oldPath + fmt.Sprintf(".margo-renaming-%d", i), filepath.Join(projectRoot, "slides", newIDs[i])})
	}
	for _, item := range plan {
		if _, err := os.Stat(item.tempPath); err == nil {
			return fmt.Errorf("temporary rename path already exists: %s", item.tempPath)
		}
	}
	for i, item := range plan {
		if err := os.Rename(item.oldPath, item.tempPath); err != nil {
			for j := i - 1; j >= 0; j-- {
				_ = os.Rename(plan[j].tempPath, plan[j].oldPath)
			}
			return err
		}
	}
	for i, item := range plan {
		if err := os.Rename(item.tempPath, item.newPath); err != nil {
			for j := i - 1; j >= 0; j-- {
				_ = os.Rename(plan[j].newPath, plan[j].oldPath)
			}
			for j := i; j < len(plan); j++ {
				_ = os.Rename(plan[j].tempPath, plan[j].oldPath)
			}
			return err
		}
	}
	return nil
}

func rewriteSlideOrders(projectRoot string, ids []string) error {
	for i, id := range ids {
		path := filepath.Join(projectRoot, "slides", id, "index.md")
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(data)
		if !strings.HasPrefix(source, "---\n") {
			updated := fmt.Sprintf("---\norder: %d\n---\n%s", i+1, source)
			if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
				return err
			}
			continue
		}
		frontMatterEnd := strings.Index(source[4:], "\n---\n")
		if frontMatterEnd < 0 {
			return fmt.Errorf("%s has unclosed front matter", path)
		}
		frontMatterEnd += 4
		frontMatter := source[:frontMatterEnd]
		lines := strings.Split(frontMatter, "\n")
		foundOrder := false
		for lineIndex, line := range lines {
			if strings.HasPrefix(line, "order:") {
				lines[lineIndex] = fmt.Sprintf("order: %d", i+1)
				foundOrder = true
			}
		}
		if !foundOrder {
			lines = append(lines, fmt.Sprintf("order: %d", i+1))
		}
		updated := strings.Join(lines, "\n") + source[frontMatterEnd:]
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func runNewNote(args []string, stdout io.Writer) error {
	noteName, slideID, err := parseNewNoteArgs(args)
	if err != nil {
		return err
	}
	if noteName == "" {
		return fmt.Errorf("new note requires a note name")
	}
	if slideID == "" {
		return fmt.Errorf("new note requires --slide <slide-bundle>")
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
			message: "new note requires a Margo project root",
			report:  report,
		}
	}

	path, err := scaffold.CreateNote(scaffold.NoteOptions{
		ProjectRoot: root.Dir,
		Slide:       slideID,
		Name:        noteName,
	})
	if err != nil {
		return fmt.Errorf("create note scaffold: %w", err)
	}
	fmt.Fprintf(stdout, "created note at %s\n", path)
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

func runThemeAdd(args []string, stdout io.Writer) error {
	repo, ref, name, err := parseThemeAddArgs(args)
	if err != nil {
		return err
	}
	if repo == "" {
		return fmt.Errorf("theme add requires a git repo")
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
			message: "theme add requires a Margo project root",
			report:  report,
		}
	}

	installed, err := theme.Install(theme.InstallOptions{
		ProjectRoot: root.Dir,
		Repo:        repo,
		Ref:         ref,
		Name:        name,
	})
	if err != nil {
		return fmt.Errorf("install theme: %w", err)
	}

	fmt.Fprintf(stdout, "installed theme %s at %s\n", installed.Name, filepath.Join(root.Dir, "themes", installed.Name))
	if installed.Source != nil {
		fmt.Fprintf(stdout, "theme source: %s %s\n", installed.Source.Type, installed.Source.Repo)
		if installed.Source.ResolvedRef != "" {
			fmt.Fprintf(stdout, "theme revision: %s\n", installed.Source.ResolvedRef)
		}
	}
	return nil
}

func runThemePack(args []string, stdout io.Writer) error {
	selected, outputPath, err := parseThemePackArgs(args)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	root, err := project.Discover(wd)
	if err != nil {
		return fmt.Errorf("theme pack requires a Margo project root: %w", err)
	}
	if selected == "" {
		selected, err = chooseThemeForPack(root.Dir, os.Stdin, stdout, isInteractiveStdin(os.Stdin))
		if err != nil {
			return err
		}
	}
	if outputPath == "" {
		outputPath = filepath.Join(filepath.Dir(root.Dir), selected+themearchive.Extension)
	} else if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(wd, outputPath)
	}
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve theme archive output: %w", err)
	}
	manifest, err := themearchive.Pack(root.Dir, selected, outputPath, version.Current())
	if err != nil {
		return fmt.Errorf("pack theme archive: %w", err)
	}
	fmt.Fprintf(stdout, "packed theme %s %s at %s\n", manifest.ThemeName, manifest.ThemeVersion, outputPath)
	return nil
}

func runThemeImport(args []string, stdout io.Writer) error {
	archivePath, localName, activate, err := parseThemeImportArgs(args)
	if err != nil {
		return err
	}
	if archivePath == "" {
		return fmt.Errorf("theme import requires a .margot archive")
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	root, err := project.Discover(wd)
	if err != nil {
		return fmt.Errorf("theme import requires a Margo project root: %w", err)
	}
	if !filepath.IsAbs(archivePath) {
		archivePath = filepath.Join(wd, archivePath)
	}
	installed, err := themearchive.Import(root.Dir, archivePath, localName, version.Current())
	if err != nil {
		return fmt.Errorf("import theme archive: %w", err)
	}
	if activate {
		if err := config.SetThemeName(root.ConfigPath, installed.Name); err != nil {
			_ = os.RemoveAll(filepath.Join(root.Dir, theme.ThemesDirName, installed.Name))
			return fmt.Errorf("activate imported theme: %w", err)
		}
	}
	fmt.Fprintf(stdout, "imported theme %s at %s\n", installed.Name, filepath.Join(root.Dir, theme.ThemesDirName, installed.Name))
	if activate {
		fmt.Fprintf(stdout, "activated theme %s\n", installed.Name)
	}
	return nil
}

func runThemeList(stdout io.Writer) error {
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
			message: "theme list requires a Margo project root",
			report:  report,
		}
	}

	themes, err := theme.List(root.Dir)
	if err != nil {
		return fmt.Errorf("list themes: %w", err)
	}
	if len(themes) == 0 {
		fmt.Fprintln(stdout, "no themes found")
		return nil
	}
	for _, installed := range themes {
		if installed.Source != nil && installed.Source.Type == "archive" {
			label := "archive"
			if installed.Source.ImportedThemeVersion != "" {
				label += " " + installed.Source.ImportedThemeVersion
			}
			if installed.Source.ArchiveSHA256 != "" {
				label += " @ " + installed.Source.ArchiveSHA256[:min(12, len(installed.Source.ArchiveSHA256))]
			}
			fmt.Fprintf(stdout, "%s - %s\n", installed.Name, label)
			continue
		}
		if installed.Source != nil && strings.TrimSpace(installed.Source.Repo) != "" {
			suffix := installed.Source.Repo
			if strings.TrimSpace(installed.Source.ResolvedRef) != "" {
				suffix += " @ " + installed.Source.ResolvedRef
			}
			fmt.Fprintf(stdout, "%s - %s\n", installed.Name, suffix)
			continue
		}
		fmt.Fprintln(stdout, installed.Name)
	}
	return nil
}

func runThemeUpdate(args []string, stdout io.Writer) error {
	themeName, err := parseThemeUpdateArgs(args)
	if err != nil {
		return err
	}
	if themeName == "" {
		return fmt.Errorf("theme update requires a theme name")
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
			message: "theme update requires a Margo project root",
			report:  report,
		}
	}

	updated, err := theme.Update(root.Dir, themeName)
	if err != nil {
		return fmt.Errorf("update theme: %w", err)
	}

	fmt.Fprintf(stdout, "updated theme %s at %s\n", updated.Name, filepath.Join(root.Dir, "themes", updated.Name))
	if updated.Source != nil {
		fmt.Fprintf(stdout, "theme source: %s %s\n", updated.Source.Type, updated.Source.Repo)
		if updated.Source.ResolvedRef != "" {
			fmt.Fprintf(stdout, "theme revision: %s\n", updated.Source.ResolvedRef)
		}
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

	includeDrafts, openBrowser, servePort, err := parseBuildLikeArgs(name, args)
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
		renderPNG := name == "build" && parsed.Config.Outputs.PNG
		renderPPTX := name == "build" && parsed.Config.Outputs.PPTX
		if parsed.Config.Outputs.HTML {
			report, err := html.Write(root.Dir, model, activeTheme)
			if err != nil {
				return err
			}
			if len(report.Items) > 0 {
				diagnostics.WriteReport(stdout, report)
			}
		}
		if renderPDF || renderPNG {
			report, err := printhtml.Write(root.Dir, model, activeTheme)
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
		if renderPNG {
			if err := png.Write(root.Dir, model.Slides); err != nil {
				return err
			}
		}
		if renderPPTX {
			report, err := pptx.Write(root.Dir, model, activeTheme)
			if err != nil {
				return err
			}
			if len(report.Items) > 0 {
				diagnostics.WriteReport(stdout, report)
			}
		}
		return nil
	}

	if name == "build" && (parsed.Config.Outputs.HTML || parsed.Config.Outputs.PDF || parsed.Config.Outputs.PNG || parsed.Config.Outputs.PPTX) {
		if err := rebuild(); err != nil {
			return fmt.Errorf("build outputs: %w", err)
		}
		filteredSlides, err := resolveSlides(includeDrafts)
		if err != nil {
			return fmt.Errorf("resolve filtered slides: %w", err)
		}
		filteredSlides = deck.ApplySectionDividers(filteredSlides)
		fmt.Fprintf(stdout, "%s: rendering %d slides after filtering\n", name, len(filteredSlides))
		if parsed.Config.Outputs.HTML {
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, html.OutputFile)
		}
		if parsed.Config.Outputs.PDF {
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, printhtml.OutputFile)
			if browser, browserErr := pdf.DetectBrowser(); browserErr == nil {
				fmt.Fprintf(stdout, "%s: pdf browser %s (%s)\n", name, browser.Path, browser.Source)
			}
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, pdf.OutputFile)
		}
		if parsed.Config.Outputs.PNG {
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, png.OutputDir)
		}
		if parsed.Config.Outputs.PPTX {
			fmt.Fprintf(stdout, "%s: wrote %s\n", name, pptx.OutputFile)
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
			modelSlides, err := resolveSlides(includeDrafts)
			if err != nil {
				return err
			}
			modelSlides = deck.ApplySectionDividers(modelSlides)
			model := deck.Model{
				Config:   parsed.Config,
				Slides:   modelSlides,
				Sections: deck.BuildSections(modelSlides),
			}
			activeTheme, err := theme.Load(root.Dir, parsed.Config.Theme.Name)
			if err != nil {
				return err
			}
			parsed.Config.Theme.Options, err = theme.ResolveOptions(activeTheme, parsed.Config.Theme.Options)
			if err != nil {
				return err
			}
			model.Config.Theme.Options = parsed.Config.Theme.Options
			if _, err := printhtml.Write(root.Dir, model, activeTheme); err != nil {
				return err
			}
			return pdf.Write(root.Dir)
		}
		return serve.Start(root.Dir, rebuild, serve.Options{
			OpenBrowser: openBrowser,
			Port:        servePort,
			Input:       os.Stdin,
			Output:      stdout,
			Interactive: isInteractiveStdin(os.Stdin),
			PDFEnabled:  parsed.Config.Outputs.PDF,
			PDFPath:     filepath.Join(root.Dir, pdf.OutputFile),
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
	fmt.Fprintf(w, "%s %s\n\n", version.Name, version.Current())
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  margo <command> [arguments]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  build        Build all configured outputs for the current deck")
	fmt.Fprintln(w, "  serve        Serve the current deck locally")
	fmt.Fprintln(w, "  pack         Package a deck project as a portable .margo archive")
	fmt.Fprintln(w, "  unpack       Restore a portable .margo archive to a project folder")
	fmt.Fprintln(w, "  theme        Install or inspect vendored themes")
	fmt.Fprintln(w, "  slide        Insert, move, delete, and reorder slides in a deck")
	fmt.Fprintln(w, "  new          Create a deck, slide, or theme scaffold")
	fmt.Fprintln(w, "  init         Initialize a deck in the current directory")
	fmt.Fprintln(w, "  upgrade      Safely refresh Margo-managed project scaffolding")
	fmt.Fprintln(w, "  skills       Install Margo-provided agent skills")
	fmt.Fprintln(w, "  deploy       Configure a deployment workflow for the current deck")
	fmt.Fprintln(w, "  clean        Remove generated output and tool-managed build state")
	fmt.Fprintln(w, "  version      Print version information")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Common options:")
	fmt.Fprintln(w, "  margo build --include-drafts")
	fmt.Fprintln(w, "  margo serve [--include-drafts] [--no-open] [--port <port>]")
	fmt.Fprintln(w, "  margo pack <deck-dir>")
	fmt.Fprintln(w, "  margo unpack <archive.margo> [destination]")
	fmt.Fprintln(w, "  margo <archive.margo> [--include-drafts] [--no-open] [--port <port>]")
	fmt.Fprintln(w, "  margo theme add <repo> [--ref <rev>] [--name <local-name>]")
	fmt.Fprintln(w, "  margo theme pack [<theme-name> | --theme <name>] [--output <archive.margot>]")
	fmt.Fprintln(w, "  margo theme import <archive.margot> [--name <local-name>] [--activate]")
	fmt.Fprintln(w, "  margo theme update <name>")
	fmt.Fprintln(w, "  margo theme list")
	fmt.Fprintln(w, "  margo upgrade --plan|--apply")
	fmt.Fprintln(w, "  margo skills install brand-theme --scope user|project [--plan]")
	fmt.Fprintln(w, "  margo deploy github-pages [--workflow-name <name>] [--margo-version <version>] [--replace]")
	fmt.Fprintln(w, "  margo theme pptx init|inspect|validate <name>")
	fmt.Fprintln(w, "  margo new slide <name> [--archetype <name>]")
	fmt.Fprintln(w, "  margo slide insert <name> (--before <slide> | --after <slide> | --position <n>) [--archetype <name>] [--renumber]")
	fmt.Fprintln(w, "  margo slide move <slide> (--before <slide> | --after <slide> | --position <n>) [--renumber]")
	fmt.Fprintln(w, "  margo slide delete <slide> [--renumber]")
	fmt.Fprintln(w, "  margo new note <name> --slide <slide-bundle>")
	fmt.Fprintln(w, "  margo new theme <name> [--blank]")
}

func parseBuildLikeArgs(command string, args []string) (bool, bool, string, error) {
	includeDrafts := command == "serve"
	openBrowser := command == "serve"
	servePort := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--include-drafts":
			includeDrafts = true
		case "--no-open":
			if command != "serve" {
				return false, false, "", fmt.Errorf("%s does not support %s", command, arg)
			}
			openBrowser = false
		case "--port":
			if command != "serve" {
				return false, false, "", fmt.Errorf("%s does not support %s", command, arg)
			}
			if i+1 >= len(args) {
				return false, false, "", fmt.Errorf("serve requires a value for --port")
			}
			servePort = strings.TrimSpace(args[i+1])
			if !isValidServePort(servePort) {
				return false, false, "", fmt.Errorf("serve port %q is invalid; use a value between 1 and 65535", servePort)
			}
			i++
		default:
			return false, false, "", fmt.Errorf("unknown %s option %q", command, arg)
		}
	}

	return includeDrafts, openBrowser, servePort, nil
}

func isValidServePort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port >= 1 && port <= 65535
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

func parseNewNoteArgs(args []string) (string, string, error) {
	var noteName string
	var slideID string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--slide":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("new note requires a value for --slide")
			}
			slideID = strings.TrimSpace(args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return "", "", fmt.Errorf("unknown new note option %q", args[i])
			}
			if noteName != "" {
				return "", "", fmt.Errorf("new note accepts exactly one note name")
			}
			noteName = args[i]
		}
	}

	return noteName, slideID, nil
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

func parseThemeUpdateArgs(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	if len(args) > 1 {
		return "", fmt.Errorf("theme update accepts exactly one theme name")
	}
	if strings.HasPrefix(args[0], "--") {
		return "", fmt.Errorf("unknown theme update option %q", args[0])
	}
	return strings.TrimSpace(args[0]), nil
}

func parseThemeAddArgs(args []string) (string, string, string, error) {
	var repo string
	var ref string
	var name string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ref":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("theme add requires a value for --ref")
			}
			ref = strings.TrimSpace(args[i+1])
			i++
		case "--name":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("theme add requires a value for --name")
			}
			name = strings.TrimSpace(args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return "", "", "", fmt.Errorf("unknown theme add option %q", args[i])
			}
			if repo != "" {
				return "", "", "", fmt.Errorf("theme add accepts exactly one repo")
			}
			repo = args[i]
		}
	}

	return repo, ref, name, nil
}

func parseThemePackArgs(args []string) (string, string, error) {
	var name, output string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--theme":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("theme pack requires a value for --theme")
			}
			candidate := strings.TrimSpace(args[i+1])
			if name != "" && name != candidate {
				return "", "", fmt.Errorf("theme pack selector conflicts with --theme")
			}
			name = candidate
			i++
		case "--output":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("theme pack requires a value for --output")
			}
			output = strings.TrimSpace(args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return "", "", fmt.Errorf("unknown theme pack option %q", args[i])
			}
			if name != "" {
				return "", "", fmt.Errorf("theme pack accepts exactly one theme name")
			}
			name = strings.TrimSpace(args[i])
		}
	}
	return name, output, nil
}

func parseThemeImportArgs(args []string) (string, string, bool, error) {
	var archivePath, name string
	activate := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 >= len(args) {
				return "", "", false, fmt.Errorf("theme import requires a value for --name")
			}
			name = strings.TrimSpace(args[i+1])
			i++
		case "--activate":
			activate = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return "", "", false, fmt.Errorf("unknown theme import option %q", args[i])
			}
			if archivePath != "" {
				return "", "", false, fmt.Errorf("theme import accepts exactly one archive")
			}
			archivePath = args[i]
		}
	}
	return archivePath, name, activate, nil
}

func chooseThemeForPack(projectRoot string, input io.Reader, stdout io.Writer, interactive bool) (string, error) {
	available, err := theme.List(projectRoot)
	if err != nil {
		return "", fmt.Errorf("list themes: %w", err)
	}
	if len(available) == 0 {
		return "", errors.New("theme pack found no installed themes")
	}
	if !interactive {
		var names []string
		for _, installed := range available {
			names = append(names, installed.Name)
		}
		return "", fmt.Errorf("theme pack requires a theme name; available themes: %s", strings.Join(names, ", "))
	}
	fmt.Fprintln(stdout, "choose a theme to package:")
	activeName := ""
	if raw, loadErr := config.LoadRaw(filepath.Join(projectRoot, config.DefaultFilename)); loadErr == nil {
		if parsed, parseErr := config.Parse(raw); parseErr == nil {
			activeName = parsed.Config.Theme.Name
		}
	}
	for i, installed := range available {
		label := installed.Name
		if installed.Name == activeName {
			label += " (active)"
		}
		fmt.Fprintf(stdout, "  %d. %s\n", i+1, label)
	}
	reader := bufio.NewReader(input)
	for {
		fmt.Fprint(stdout, "select theme: ")
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return "", fmt.Errorf("read theme selection: %w", readErr)
		}
		choice, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr == nil && choice >= 1 && choice <= len(available) {
			return available[choice-1].Name, nil
		}
		if errors.Is(readErr, io.EOF) {
			return "", fmt.Errorf("invalid theme selection %q", strings.TrimSpace(line))
		}
		fmt.Fprintln(stdout, "invalid selection; enter a number from the list")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
