package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/a-novel-kit/configurator/chans"
	"github.com/a-novel-kit/configurator/utilstest"
	"github.com/a-novel-kit/context"

	"github.com/a-novel/service-authentication/api/apiclient/testapiclient"
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

	go func() {
		listener := logs.Register()
		for msg := range listener {
			// Forward logs to default system outputs, in case we need them for debugging.
			log.Println(msg)
		}
	}()
}

func _runKeysRotation() {
	// Run keys rotation.
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	rotateKeysPath := path.Join(filepath.Dir(pwd), "..", "cmd", "rotatekeys", "main.go")

	out, err := exec.CommandContext(context.Background(), "go", "run", rotateKeysPath).CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("rotate keys: %v\n%v", err, string(out)))
	}
}

// Create a separate database to run integration tests.
func init() {
	_patchSTD()

	_runKeysRotation()

	go func() {
		main()
	}()

	_, _, err := testapiclient.GetServerClient()
	if err != nil {
		panic(err)
	}
}
