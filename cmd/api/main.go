package main

import (
	"context"
	"log"

	cmdpkg "github.com/a-novel/service-authentication/pkg/cmd"
)

func main() {
	err := cmdpkg.App(context.Background(), cmdpkg.AppConfigDefault)
	if err != nil {
		log.Fatalf("initialize app: %v", err)
	}
}
