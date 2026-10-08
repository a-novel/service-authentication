// Command maintenance runs trusted, one-shot authentication maintenance operations.
// Operations are selected by subcommand so new routines can be added without changing
// the invocation model. Running the image without an operation only displays help.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

func main() {
	os.Exit(mainExitCode())
}

func mainExitCode() int {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("maintenance: ")

	// Every operation works on the database, which is unavailable during a planned downtime.
	if config.AppPresetDefault.App.Downtime.InProgress(config.DowntimeService, time.Now()) {
		log.Print("planned downtime in progress: nothing done")

		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	command := newMaintenanceCommand(accountRoleOperationDefinition(), waitlistCleanupOperationDefinition())

	err := command.run(ctx, os.Args[1:], os.Stdout)
	if err != nil {
		log.Print(err)

		return 1
	}

	return 0
}
