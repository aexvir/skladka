package main

import (
	"context"

	"github.com/aexvir/harness/commons"
)

// test the whole codebase
func Test(ctx context.Context) error {
	return h.Execute(
		ctx,
		commons.GoTest(
			commons.WithTestJunit(commons.IsCIEnv()),
			commons.WithTestCobertura(commons.IsCIEnv()),
			commons.WithTestCIFriendlyOutput(commons.IsCIEnv()),
		),
	)
}
