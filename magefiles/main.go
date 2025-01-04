package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aexvir/harness"
	"github.com/aexvir/harness/bintool"
	"github.com/aexvir/harness/commons"

	"github.com/aexvir/skladka/internal/errors"
)

const (
	pkgName             = "github.com/aexvir/skladka"
	commitsarVersion    = "0.20.1"
	golangcilintVersion = "v1.62.2"
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
	return h.Execute(
		ctx,
		commons.OnlyLocally(commons.GoBuild("./cmd", "bin/skladka")),
		commons.OnlyOnCI(
			commons.GoBuild("./cmd", "bin/skladka",
				commons.WithGoBuildTags("osusergo", "netgo"),
				commons.WithGoBuildLDFlags(
					fmt.Sprintf("%s/internal/config.BuildBranch=%s", pkgName, os.Getenv("BUILD_BRANCH")),
					fmt.Sprintf("%s/internal/config.BuildRevision=%s", pkgName, os.Getenv("BUILD_REV")),
					fmt.Sprintf("%s/internal/config.BuildDate=%s", pkgName, time.Now().Format(time.RFC3339)),
				),
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
	air, _ := bintool.NewGo(
		"github.com/air-verse/air",
		"latest",
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

func ptr[t any](item t) *t {
	return &item
}
