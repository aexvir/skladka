package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aexvir/harness"
	"github.com/aexvir/harness/bintool"
	"github.com/aexvir/harness/commons"

	"github.com/aexvir/skladka/internal/errors"
)

// generate code and static files
func Generate(ctx context.Context) error {
	return h.Execute(
		ctx,
		// run go generate ./...
		commons.GoGenerate(),
		// generate sql code
		func(ctx context.Context) error {
			sqlc, _ := bintool.NewGo(
				"github.com/sqlc-dev/sqlc/cmd/sqlc",
				"latest",
			)

			if err := sqlc.Ensure(); err != nil {
				return errors.Wrap(err, "failed to provision sqlc")
			}

			return harness.Run(ctx, sqlc.BinPath(), harness.WithArgs("generate"))
		},
		// generate templates
		func(ctx context.Context) error {
			templ, _ := bintool.NewGo(
				"github.com/a-h/templ/cmd/templ",
				"latest",
			)

			if err := templ.Ensure(); err != nil {
				return errors.Wrap(err, "failed to provision templ")
			}

			return harness.Run(
				ctx,
				templ.BinPath(),
				harness.WithArgs("generate", "-include-version=false"),
			)
		},
		// generate documentation
		func(ctx context.Context) error {
			gomarkdoc, _ := bintool.NewGo(
				"github.com/princjef/gomarkdoc/cmd/gomarkdoc",
				"latest",
			)

			if err := gomarkdoc.Ensure(); err != nil {
				return fmt.Errorf("failed to provision gomarkdoc: %w", err)
			}

			// ensure docs/content directory exists
			if err := os.MkdirAll("docs/content", 0o755); err != nil {
				return fmt.Errorf("failed to create docs directory: %w", err)
			}

			// find all go packages in the project
			packages, err := findGoPackages(".", "docs", "infra", "magefiles")
			if err != nil {
				return fmt.Errorf("failed to find go packages: %w", err)
			}

			// generate documentation for each package
			for _, pkg := range packages {
				output := getOutputPath(pkg)
				if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %w", pkg, err)
				}

				if err := harness.Run(
					ctx,
					gomarkdoc.BinPath(),
					harness.WithArgs("--output", output, "./"+pkg),
				); err != nil {
					return fmt.Errorf("failed to generate docs for %s: %w", pkg, err)
				}
			}

			return nil
		},
	)
}

// findGoPackages walks the directory tree and returns a list of Go package paths
func findGoPackages(root string, skipdirs ...string) ([]string, error) {
	var packages []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		// Skip directories we don't want to document
		dirname := filepath.Base(path)
		for _, skip := range skipdirs {
			if dirname == skip {
				return filepath.SkipDir
			}
		}

		// Check if directory contains Go files
		hasGoFiles := false

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
				hasGoFiles = true
				break
			}
		}

		if hasGoFiles {
			relpath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if relpath != "." {
				packages = append(packages, relpath)
			}
		}

		return nil
	})

	return packages, err
}

// getOutputPath determines the output path for a package's documentation
func getOutputPath(pkg string) string {
	base := filepath.Join("docs/content/docs", pkg)

	// check if it's a leaf package or an intermediary package
	isLeafPackage := true
	filepath.Walk(
		pkg,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && path != pkg {
				isLeafPackage = false
				return filepath.SkipDir
			}

			return nil
		},
	)

	if isLeafPackage {
		return base + ".md"
	}

	return filepath.Join(base, "_index.md")
}
