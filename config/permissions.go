package config

import (
	_ "embed"

	"github.com/a-novel-kit/configurator"

	"github.com/a-novel/service-authentication/models"
)

//go:embed permissions.yaml
var permissionsFile []byte

var Permissions = configurator.NewLoader[models.PermissionsConfig](Loader).
	MustLoad(configurator.NewConfig("", permissionsFile))
