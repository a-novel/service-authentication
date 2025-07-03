package main

import (
	"log"
	"os"

	"github.com/a-novel-kit/configurator/chans"
	"github.com/a-novel-kit/configurator/utilstest"

	"github.com/a-novel/service-authentication/internal/api/apiclient/testapiclient"
	"github.com/a-novel/service-authentication/internal/services"
)

var logs *chans.MultiChan[string]

func _patchSTD() {
	patchedStd, _, err := utilstest.MonkeyPatchStderr()
	if err != nil {
		panic(err)
	}

	logs, _, err = utilstest.CaptureSTD(patchedStd)
	if err != nil {
		panic(err)
	}

	// Patching stderr does not propagate to service, IDK why.
	services.DebugLogger = log.New(os.Stderr, "", 0)

	go func() {
		listener := logs.Register()
		for msg := range listener {
			// Forward logs to default system outputs, in case we need them for debugging.
			log.Println("forwarded:", msg)
		}
	}()
}

// Create a separate database to run integration tests.
func init() {
	_patchSTD()

	go func() {
		main()
	}()

	_, _, err := testapiclient.GetServerClient()
	if err != nil {
		panic(err)
	}
}
