package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aexvir/harness"
	"github.com/aexvir/harness/binary"
	"github.com/aexvir/harness/commons"

	"github.com/aexvir/skladka/internal/errors"
)

const (
	pkgName             = "github.com/aexvir/skladka"
	commitsarVersion    = "0.20.1"
	golangcilintVersion = "v1.64.5"
)

var h = harness.New(
	harness.WithPreExecFunc(
		func(ctx context.Context) error {
			// ensure go mod download is run before any task
			return harness.Run(ctx, "go", harness.WithArgs("mod", "download"))
		},
	),
)

// format codebase using gofmt and goimports
func Format(ctx context.Context) error {
	return h.Execute(
		ctx,
		commons.GoFmt(),
		commons.GoImports(pkgName),
	)
}

// run go mod tidy
func Tidy(ctx context.Context) error {
	return h.Execute(
		ctx,
		commons.GoModTidy(),
	)
}

// lint the code using go mod tidy, hadolint and golangci-lint
func Lint(ctx context.Context) error {
	return h.Execute(
		ctx,
		commons.GoModTidy(),
		commons.OnlyLocally(
			commons.Commitsar(
				commons.WithCommitsarVersion(commitsarVersion),
			),
		),
		commons.GolangCILint(
			commons.WithGolangCIVersion(golangcilintVersion),
			commons.WithGolangCICodeClimate(commons.IsCIEnv()),
		),
	)
}

// build the skladka binary
func Build(ctx context.Context) error {
	branch, revision, date, err := buildmeta(ctx)
	if err != nil {
		return err
	}

	return h.Execute(
		ctx,
		commons.GoBuild("./cmd", "bin/skladka",
			commons.WithGoBuildTags("osusergo", "netgo"),
			commons.WithGoBuildLDFlags(
				fmt.Sprintf("%s/internal/config.BuildBranch=%s", pkgName, branch),
				fmt.Sprintf("%s/internal/config.BuildRevision=%s", pkgName, revision),
				fmt.Sprintf("%s/internal/config.BuildDate=%s", pkgName, date),
			),
		),
		commons.OnlyLocally(
			func(ctx context.Context) error {
				return harness.Run(
					ctx,
					"open",
					harness.WithArgs("raycast://extensions/raycast/raycast/confetti"),
					harness.WithoutNoise(),
					harness.WithAllowErrors(),
				)
			},
		),
	)
}

// build the skladka binary, then run it
func Run(ctx context.Context) error {
	if err := Generate(ctx); err != nil {
		return err
	}

	if err := Build(ctx); err != nil {
		return err
	}

	return harness.Run(
		ctx,
		"bin/skladka",
	)
}

func Dev(ctx context.Context) error {
	// note: installing process compose with go install kinda sucks
	air, _ := binary.New(
		"air",
		"latest",
		binary.GoBinary("github.com/air-verse/air"),
	)

	if err := air.Ensure(); err != nil {
		return errors.Wrap(err, "failed to provision air")
	}

	return harness.Run(
		ctx,
		"process-compose",
		harness.WithArgs("up"),
	)
}

func buildmeta(ctx context.Context) (branch, revision, date string, err error) {
	branch, revision, date = os.Getenv("BUILD_BRANCH"), os.Getenv("BUILD_REV"), time.Now().Format(time.RFC3339)

	if branch == "" {
		var b bytes.Buffer
		err := harness.Run(ctx, "git", harness.WithArgs("branch", "--show-current"), harness.WithStdOut(&b))
		if err != nil {
			return "", "", "", errors.Wrap(err, "failed to get branch")
		}
		branch = b.String()
	}

	if revision == "" {
		var r bytes.Buffer
		err := harness.Run(ctx, "git", harness.WithArgs("rev-parse", "--short", "HEAD"), harness.WithStdOut(&r))
		if err != nil {
			return "", "", "", errors.Wrap(err, "failed to get revision")
		}

		var s bytes.Buffer
		err = harness.Run(ctx, "git", harness.WithArgs("status", "-uno", "-s"), harness.WithStdOut(&s))

		if s.Len() > 0 {
			r.WriteString("+dirty")
		}

		revision = r.String()
	}

	return branch, revision, date, nil
}

func ptr[t any](item t) *t {
	return &item
}
