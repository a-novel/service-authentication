package api

import (
	"strings"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
)

func (api *API) Ping(_ context.Context) (codegen.PingRes, error) {
	return &codegen.PingOK{Data: strings.NewReader("pong")}, nil
}
