package cmdpkg_test

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

// Test util to perform anonymous authentication. The token for the session is returned.
func authAnon(t *testing.T, appConfig TestConfig) string {
	t.Helper()

	security := pkg.NewBearerSource()
	client, err := pkg.NewAPIClient(t.Context(), fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security)
	require.NoError(t, err)

	rawRes, err := client.CreateAnonSession(t.Context())
	require.NoError(t, err)

	res, ok := rawRes.(*apimodels.Token)
	require.True(t, ok, rawRes)
	require.NotEmpty(t, res.GetAccessToken())

	return res.GetAccessToken()
}

func getShortCode(target, purpose string, appConfig TestConfig) (string, bool) {
	res, ok := appConfig.SMTP.FindTestMail(func(mail *smtp.TestMail) bool {
		log.Println(mail)

		mapData, ok := mail.Data.(map[string]any)
		if !ok {
			return false
		}

		if mapData["Target"] != target {
			return false
		}

		if mapData["_Purpose"] != purpose {
			return false
		}

		shortCode, ok := mapData["ShortCode"].(string)
		if !ok || shortCode == "" {
			return false
		}

		return true
	})

	if !ok || res == nil {
		return "", false
	}

	mapData, ok := res.Data.(map[string]any)
	if !ok {
		return "", false
	}

	shortCode, ok := mapData["ShortCode"].(string)
	if !ok || shortCode == "" {
		return "", false
	}

	return shortCode, true
}

// Test util to check the session of the authenticated user. Returns the claims of the session.
func checkSession(t *testing.T, client *apimodels.Client) *apimodels.Claims {
	t.Helper()

	rawRes, err := client.CheckSession(t.Context())
	require.NoError(t, err)

	res, ok := rawRes.(*apimodels.Claims)
	require.True(t, ok, rawRes)

	return res
}

type userData struct {
	email        string
	id           string
	password     string
	user         string
	token        string
	refreshToken string
}

func createUser(t *testing.T, appConfig TestConfig) *userData {
	t.Helper()

	security := pkg.NewBearerSource()
	client, err := pkg.NewAPIClient(t.Context(), fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security)
	require.NoError(t, err)

	token := authAnon(t, appConfig)
	security.SetToken(token)

	user := rand.Text()
	email := user + "@example.com"
	password := rand.Text()

	var shortCode, refreshToken string

	t.Log("RequestRegistration")
	{
		rawRes, err := client.RequestRegistration(t.Context(), &apimodels.RequestRegistrationForm{
			Email: apimodels.Email(email),
		})
		require.NoError(t, err)
		require.IsType(t, &apimodels.RequestRegistrationNoContent{}, rawRes)

		var ok bool

		require.Eventually(t, func() bool {
			shortCode, ok = getShortCode(base64.RawURLEncoding.EncodeToString([]byte(email)), "register", appConfig)

			return assert.True(t, ok)
		}, 10*time.Second, 100*time.Millisecond)
	}

	t.Log("CreateUser")
	{
		rawRes, err := client.Register(t.Context(), &apimodels.RegisterForm{
			Email:     apimodels.Email(email),
			Password:  apimodels.Password(password),
			ShortCode: apimodels.ShortCode(shortCode),
		})
		require.NoError(t, err)

		res, ok := rawRes.(*apimodels.Token)
		require.True(t, ok, rawRes)
		require.NotEmpty(t, res.GetAccessToken())
		require.NotEmpty(t, res.GetRefreshToken())

		token = res.GetAccessToken()
		refreshToken = res.GetRefreshToken()
	}

	security.SetToken(token)

	var userID string

	t.Log("DecodeClaims")
	{
		rawClaims, err := client.CheckSession(t.Context())
		require.NoError(t, err)
		require.NotNil(t, rawClaims)

		claims, ok := rawClaims.(*apimodels.Claims)
		require.True(t, ok, rawClaims)
		require.NotEmpty(t, claims.GetUserID())

		userID = claims.GetUserID().Value.String()
	}

	return &userData{
		email:        email,
		password:     password,
		user:         user,
		token:        token,
		refreshToken: refreshToken,
		id:           userID,
	}
}
