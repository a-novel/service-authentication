---
outline: deep
---

# Go module

The exported methods are available under the `/pkg` component.

```bash
go get github.com/a-novel/service-authentication
```

## Setup

You must provide an interface to handle claims validation. You can use the
[JSON Keys client package](https://github.com/a-novel/service-json-keys) for this purpose.

```go
package main

import (
	jkPkg "github.com/a-novel/service-json-keys/pkg"
	authModels "github.com/a-novel/service-authentication/models"
)

// serverURL := "http://localhost:4001/v1"
client, err := jkPkg.NewAPIClient(serverURL)

verifier, err := jkPkg.NewClaimsVerifier[authModels.AccessTokenClaims](client)
```

You must then plug the security handler to your API. The API must be compatible with
[OpenAPI v3 standards](https://swagger.io/specification/); more specifically, it should expose
an operation name (unique name for the current route), and an auth object.

Here is an example using [Ogen](https://github.com/ogen-go/ogen):

```go
package main

import (
	"context"
	"fmt"
	authPkg "github.com/a-novel/service-authentication/pkg"
	authModels "github.com/a-novel/service-authentication/models"

	"myproject/api/codegen"
)

type SecurityHandler struct {
	handler *authPkg.HandleBearerAuth[codegen.OperationName]
}

func NewSecurity(
	source authPkg.AuthenticateSource, permissions authModels.PermissionsConfig,
) (*SecurityHandler, error) {
	handler, err := authPkg.NewHandleBearerAuth[codegen.OperationName](source, permissions)
	if err != nil {
		return nil, fmt.Errorf("NewSecurity: %w", err)
	}

	return &SecurityHandler{handler: handler}, nil
}

func (security *SecurityHandler) HandleBearerAuth(
	ctx context.Context, operationName codegen.OperationName, auth codegen.BearerAuth,
) (context.Context, error) {
	return security.handler.HandleBearerAuth(ctx, operationName, &auth)
}
```

The `PermissionsConfig` is a map of operation names to permissions required for such operation. The
required role for a given operation are provided through the `GetRoles` method of the auth token, and
come from the OpenAPI spec (`security.bearerAuth.roles[]`)`). The object you provide matches
known user roles with a set of permissions. For example:

```go
config := authModels.PermissionsConfig{
	Roles: map[authModels.Role]authModels.RoleConfig{
		authModels.RoleAnon: {
			Permissions: []authModels.Permission{
				"user:read",
			},
		},
		authModels.RoleUser: {
			Inherits: []authModels.Role{authModels.RoleAnon},
			Permissions: []authModels.Permission{
				"user:write",
				"user:delete",
			},
		},
	},
}
```

## Retrieve claims

Once a user is authenticated, claims can be retrieved from the context.

```go
claims, err := pkg.GetClaimsContext(ctx)
```
