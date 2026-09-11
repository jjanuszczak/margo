// Package deploy creates transparent GitHub Pages automation for a Margo deck.
package deploy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PagesOptions struct {
	WorkflowName string
	MargoVersion string
	Replace      bool
}

type PagesResult struct {
	WorkflowPath string
	NoJekyllPath string
}

func GitHubPages(root string, options PagesOptions) (PagesResult, error) {
	if err := isGitRepository(root); err != nil {
		return PagesResult{}, err
	}
	name := strings.TrimSpace(options.WorkflowName)
	if name == "" {
		name = "margo-pages.yml"
	}
	if filepath.Base(name) != name || !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
		return PagesResult{}, errors.New("workflow name must be a YAML filename")
	}
	version := strings.TrimSpace(options.MargoVersion)
	if version == "" {
		version = "latest"
	}
	workflow := filepath.Join(root, ".github", "workflows", name)
	if _, err := os.Stat(workflow); err == nil && !options.Replace {
		return PagesResult{}, fmt.Errorf("GitHub Pages workflow already exists: %s (use --replace to overwrite it)", workflow)
	} else if err != nil && !os.IsNotExist(err) {
		return PagesResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		return PagesResult{}, err
	}
	if err := os.WriteFile(workflow, []byte(workflowTemplate(version)), 0o644); err != nil {
		return PagesResult{}, err
	}
	noJekyll := filepath.Join(root, ".nojekyll")
	if _, err := os.Stat(noJekyll); os.IsNotExist(err) {
		if err := os.WriteFile(noJekyll, nil, 0o644); err != nil {
			return PagesResult{}, err
		}
	} else if err != nil {
		return PagesResult{}, err
	}
	return PagesResult{WorkflowPath: workflow, NoJekyllPath: noJekyll}, nil
}

func isGitRepository(root string) error {
	command := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree")
	output, err := command.Output()
	if err != nil || strings.TrimSpace(string(output)) != "true" {
		return errors.New("github-pages deployment requires the deck to be in a Git repository")
	}
	return nil
}

func workflowTemplate(version string) string {
	return fmt.Sprintf(`name: Deploy Margo deck to GitHub Pages

on:
  push:
    tags: ['v*']
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: github-pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          # Deck repositories do not need their own go.mod. Keep this aligned
          # with Margo's supported Go toolchain instead.
          go-version: '1.22.x'
          cache: true
      - name: Install Margo
        run: go install github.com/jjanuszczak/margo/cmd/margo@%s
      - name: Build deck
        run: margo build
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: dist/html
  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
`, version)
}
