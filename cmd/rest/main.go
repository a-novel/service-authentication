// Command rest runs the authentication service's HTTP API. It wires the data-access, core,
// and handler layers together, mounts them on a chi router, and serves until an interrupt or
// termination signal arrives, then shuts the server down gracefully.
//
// The server admits only the callers its service tokens allow, and refuses to start without a token
// issuer unless SERVICE_ISSUER_OPTIONAL opts a local stack out.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
	"github.com/a-novel/service-authentication/v2/pkg/go"
)

func main() {
	cfg := config.AppPresetDefault

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	// =================================================================================================================
	// DEPENDENCIES
	// =================================================================================================================

	// A server started during a planned downtime refuses work, so it leaves the database alone.
	if !downtime.Started(cfg.App.DowntimeStart, time.Now()) {
		ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))
		defer cfg.Postgres.Close()
	}

	lo.Must0(cfg.OAuth.Check(cfg.ServiceToken))

	jsonKeys := cfg.Dependencies.ServiceJSONKeys

	jsonKeysClient := lo.Must(servicejsonkeys.NewClient(jsonKeys.Address(), jsonKeys.DialOptions()...))
	defer jsonKeysClient.Close()

	mailDelivery := core.NewMailDelivery(cfg.SMTP, env.SMTPMaxConcurrent)
	lib.LimitArgon2(cfg.Limits.Argon2MaxConcurrent)
	lib.PrepareDummyArgon2()

	// Missing SMTP_ADDR looks like healthy mail loss: 202 answered, mail rendered to stdout.
	if _, debug := cfg.SMTP.(*smtp.DebugSender); debug {
		log.Println("WARNING: SMTP_ADDR is not set; emails are printed to stdout by the debug sender and none are delivered")
	}

	router := lo.Must(newRouter(cfg, jsonKeysClient, mailDelivery))

	lo.Must0(serve(ctx, stop, cfg.Rest, router, mailDelivery))
}

// newRouter builds the REST API on the JSON Keys client and the mail delivery main owns. Each request
// passes the service-token gate first, but for the liveness probe. The gate refuses an empty issuer
// unless cfg opts a local stack out, so the server never starts open by mistake.
//
//nolint:maintidx // newRouter wires the whole dependency graph as one flat list, one line per dependency.
func newRouter(
	cfg config.App, jsonKeysClient servicejsonkeys.Client, mailDelivery *core.MailDelivery,
) (http.Handler, error) {
	withServiceToken, err := httpf.ServiceTokenGate(
		cfg.ServiceToken, httpf.NewPoolClient(httpf.PoolOptions{}), cfg.Logger, cfg.Rest.Liveness,
	)
	if err != nil {
		return nil, fmt.Errorf("service token gate: %w", err)
	}

	serviceVerifyAccessToken, err := servicejsonkeys.NewClaimsVerifier[session.Claims](jsonKeysClient)
	if err != nil {
		return nil, fmt.Errorf("access token verifier: %w", err)
	}

	serviceVerifyRefreshToken, err := servicejsonkeys.NewClaimsVerifier[session.RefreshClaims](jsonKeysClient)
	if err != nil {
		return nil, fmt.Errorf("refresh token verifier: %w", err)
	}

	refreshTokenLeeway := jsonKeysClient.Keys()[servicejsonkeys.KeyUsageAuthRefresh].Token.Leeway
	accessTokenTTL := jsonKeysClient.Keys()[servicejsonkeys.KeyUsageAuth].Token.TTL

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoWaitlist, err := dao.NewGoogleWaitlist(cfg.Waitlist, nil)
	if err != nil {
		return nil, fmt.Errorf("waitlist writer: %w", err)
	}

	daoShortCodeDelete := dao.NewPgShortCodeDelete()
	daoShortCodeInsert := dao.NewPgShortCodeInsert()
	daoShortCodeSelect := dao.NewPgShortCodeSelect()

	daoCredentialsExist := dao.NewPgCredentialsExist()
	daoTransactor := postgres.NewTransactor(nil)

	daoCredentialsInsert := dao.NewPgCredentialsInsert()
	daoCredentialsList := dao.NewPgCredentialsList()
	daoCredentialsSelect := dao.NewPgCredentialsSelect()
	daoCredentialsSelectByEmail := dao.NewPgCredentialsSelectByEmail()
	daoCredentialsUpdateEmail := dao.NewPgCredentialsUpdateEmail()
	daoCredentialsUpdatePassword := dao.NewPgCredentialsUpdatePassword()
	daoCredentialsUpdateRole := dao.NewPgCredentialsUpdateRole()
	daoCredentialsUpdateLocationConsent := dao.NewPgCredentialsUpdateLocationConsent()
	daoCredentialsLocationConsent := dao.NewPgCredentialsLocationConsent()

	daoRefreshTokenRevocationExist := dao.NewPgRefreshTokenRevocationExist()
	daoRefreshTokenRevocationInsert := dao.NewPgRefreshTokenRevocationInsert()

	daoSessionClearLocations := dao.NewPgSessionClearLocations()
	daoSessionDelete := dao.NewPgSessionDelete()
	daoSessionList := dao.NewPgSessionList()
	daoSessionRevokeOthers := dao.NewPgSessionRevokeOthers()
	daoSessionSelect := dao.NewPgSessionSelect()
	daoSessionTouch := dao.NewPgSessionTouch()
	daoSessionTrim := dao.NewPgSessionTrim()
	daoSessionRevokeOldest := dao.NewPgSessionRevokeOldest()
	daoSessionUpsert := dao.NewPgSessionUpsert()

	daoAuthorizationCodeConsume := dao.NewPgAuthorizationCodeConsume()
	daoAuthorizationCodeInsert := dao.NewPgAuthorizationCodeInsert()
	daoAuthorizationCodeSelect := dao.NewPgAuthorizationCodeSelect()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceShortCodeVerify := core.NewShortCodeVerify(daoShortCodeSelect)
	serviceShortCodeConsume := core.NewShortCodeConsume(daoShortCodeDelete)
	serviceShortCodeCreate := core.NewShortCodeCreate(daoShortCodeInsert, cfg.ShortCodes)
	serviceShortCodeCreateEmailUpdate := core.NewShortCodeCreateEmailUpdate(
		serviceShortCodeCreate, daoCredentialsExist, daoCredentialsSelect, mailDelivery, cfg.ShortCodes,
		cfg.SMTPURLs,
	)
	serviceShortCodeCreatePasswordReset := core.NewShortCodeCreatePasswordReset(
		serviceShortCodeCreate, daoCredentialsSelectByEmail, mailDelivery, cfg.ShortCodes, cfg.SMTPURLs,
	)
	serviceShortCodeCreateRegister := core.NewShortCodeCreateRegister(
		serviceShortCodeCreate, daoCredentialsExist, mailDelivery, cfg.ShortCodes, cfg.SMTPURLs,
	)
	serviceShortCodeCancelEmailUpdate := core.NewShortCodeCancelEmailUpdate(daoShortCodeDelete, daoShortCodeSelect)

	serviceCredentialsCreate := core.NewCredentialsCreate(
		daoCredentialsInsert, daoCredentialsLocationConsent, daoSessionUpsert, daoSessionRevokeOldest, daoSessionTrim,
		serviceShortCodeVerify, serviceShortCodeConsume, jsonKeysClient, daoTransactor, daoWaitlist, refreshTokenLeeway,
	)
	serviceCredentialsExist := core.NewCredentialsExist(daoCredentialsExist)
	serviceWaitlistJoin := core.NewWaitlistJoin(daoCredentialsExist, daoWaitlist)
	serviceCredentialsGet := core.NewCredentialsGet(daoCredentialsSelect)
	serviceCredentialsGetSelf := core.NewCredentialsGetSelf(daoCredentialsSelect, daoShortCodeSelect)
	serviceCredentialsList := core.NewCredentialsList(daoCredentialsList)
	serviceCredentialsUpdateEmail := core.NewCredentialsUpdateEmail(
		daoCredentialsUpdateEmail, daoCredentialsSelect, serviceShortCodeVerify, serviceShortCodeConsume, mailDelivery,
		daoTransactor,
	)
	serviceCredentialsUpdatePassword := core.NewCredentialsUpdatePassword(
		daoCredentialsUpdatePassword, daoCredentialsSelect, serviceShortCodeVerify, serviceShortCodeConsume, daoTransactor,
	)
	serviceCredentialsUpdateRole := core.NewCredentialsUpdateRole(daoCredentialsUpdateRole, daoCredentialsSelect)
	serviceCredentialsUpdateLocationConsent := core.NewCredentialsUpdateLocationConsent(
		daoCredentialsUpdateLocationConsent, daoSessionClearLocations, daoTransactor,
	)

	serviceClaimsGet := core.NewClaimsGet(daoRefreshTokenRevocationExist)

	serviceTokenCreate := core.NewTokenCreate(
		daoCredentialsSelectByEmail, daoCredentialsLocationConsent, daoSessionUpsert, daoSessionRevokeOldest,
		daoSessionTrim, jsonKeysClient, daoTransactor, refreshTokenLeeway,
	)
	serviceTokenCreateAnon := core.NewTokenCreateAnon(jsonKeysClient)
	serviceTokenRefresh := core.NewTokenRefresh(
		daoCredentialsSelect, daoRefreshTokenRevocationExist, daoCredentialsLocationConsent, daoSessionUpsert,
		jsonKeysClient, serviceVerifyAccessToken, serviceVerifyRefreshToken, daoTransactor,
	)
	serviceTokenRevoke := core.NewTokenRevoke(
		daoRefreshTokenRevocationInsert, serviceVerifyRefreshToken, refreshTokenLeeway,
	)

	serviceSessionList := core.NewSessionList(daoSessionList)
	serviceSessionRevoke := core.NewSessionRevoke(daoSessionSelect, daoRefreshTokenRevocationInsert, refreshTokenLeeway)
	serviceSessionRevokeOthers := core.NewSessionRevokeOthers(daoSessionRevokeOthers, refreshTokenLeeway)
	serviceSessionDelete := core.NewSessionDelete(daoSessionDelete, daoSessionSelect)

	serviceAuthorizationRequestValidate := core.NewAuthorizationRequestValidate(cfg.OAuth)
	serviceAuthorizationCodeCreate := core.NewAuthorizationCodeCreate(
		daoAuthorizationCodeInsert, daoRefreshTokenRevocationExist, cfg.OAuth,
	)
	serviceTokenGrant := core.NewTokenGrant(
		daoAuthorizationCodeSelect, daoAuthorizationCodeConsume, daoCredentialsSelect, daoRefreshTokenRevocationExist,
		daoRefreshTokenRevocationInsert, daoSessionTouch, jsonKeysClient, serviceVerifyRefreshToken, cfg.OAuth,
		accessTokenTTL, refreshTokenLeeway,
	)

	// =================================================================================================================
	// MIDDLEWARES
	// =================================================================================================================

	withSessionAuth := serviceauthentication.NewAuthHandler(serviceVerifyAccessToken, cfg.Permissions, cfg.Logger)
	refuseClientTokens := middlewares.RestRefuseClientTokens(cfg.Logger)
	// An OAuth client's token reaches the session routes and its own account alone, which
	// withSessionAuth guards.
	withAuth := func(r chi.Router, permissions ...string) chi.Router {
		return withSessionAuth(r, permissions...).With(refuseClientTokens)
	}
	limit := middlewares.NewRestLimits(cfg.Rest, cfg.Limits)
	// A signed-out session keeps its access token until it expires. The routes that manage devices
	// refuse it at once, so it cannot sign the account's other sessions out in turn.
	refuseEndedSessions := handlers.NewRestRefuseEndedSessions(serviceClaimsGet, cfg.Logger).Middleware

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerHealth := handlers.NewRestHealth(jsonKeysClient, cfg.SMTPPing, cfg.Logger)

	handlerClaimsGet := handlers.NewRestClaimsGet(serviceClaimsGet, cfg.Logger)

	handlerCredentialsCreate := handlers.NewRestCredentialsCreate(serviceCredentialsCreate, cfg.Logger)
	handlerCredentialsExist := handlers.NewRestCredentialsExist(serviceCredentialsExist, cfg.Logger)
	handlerWaitlistJoin := handlers.NewRestWaitlistJoin(serviceWaitlistJoin, cfg.Logger)
	handlerCredentialsGet := handlers.NewRestCredentialsGet(serviceCredentialsGet, cfg.Logger)
	handlerCredentialsGetSelf := handlers.NewRestCredentialsGetSelf(serviceCredentialsGetSelf, cfg.Logger)
	handlerCredentialsList := handlers.NewRestCredentialsList(serviceCredentialsList, cfg.Logger)
	handlerCredentialsResetPassword := handlers.NewRestCredentialsResetPassword(
		serviceCredentialsUpdatePassword, cfg.Logger,
	)
	handlerCredentialsUpdatePassword := handlers.NewRestCredentialsUpdatePassword(
		serviceCredentialsUpdatePassword, cfg.Logger,
	)
	handlerCredentialsUpdateEmail := handlers.NewRestCredentialsUpdateEmail(serviceCredentialsUpdateEmail, cfg.Logger)
	handlerCredentialsUpdateRole := handlers.NewRestCredentialsUpdateRole(serviceCredentialsUpdateRole, cfg.Logger)
	handlerCredentialsUpdateLocationConsent := handlers.NewRestCredentialsUpdateLocationConsent(
		serviceCredentialsUpdateLocationConsent, cfg.Logger,
	)

	handlerShortCodeCreateEmailUpdate := handlers.NewRestShortCodeCreateEmailUpdate(
		serviceShortCodeCreateEmailUpdate, cfg.Logger,
	)
	handlerShortCodeCreatePasswordReset := handlers.NewRestShortCodeCreatePasswordReset(
		serviceShortCodeCreatePasswordReset, cfg.Logger,
	)
	handlerShortCodeCreateRegister := handlers.NewRestShortCodeCreateRegister(serviceShortCodeCreateRegister, cfg.Logger)
	handlerShortCodeCancelEmailUpdate := handlers.NewRestShortCodeCancelEmailUpdate(
		serviceShortCodeCancelEmailUpdate, cfg.Logger,
	)

	handlerTokenCreate := handlers.NewRestTokenCreate(serviceTokenCreate, cfg.Logger)
	handlerTokenCreateAnon := handlers.NewRestTokenCreateAnon(serviceTokenCreateAnon, cfg.Logger)
	handlerTokenRefresh := handlers.NewRestTokenRefresh(serviceTokenRefresh, cfg.Logger)
	handlerTokenRevoke := handlers.NewRestTokenRevoke(serviceTokenRevoke, cfg.Logger)

	handlerSessionList := handlers.NewRestSessionList(serviceSessionList, cfg.Logger)
	handlerSessionRevoke := handlers.NewRestSessionRevoke(serviceSessionRevoke, cfg.Logger)
	handlerSessionRevokeOthers := handlers.NewRestSessionRevokeOthers(serviceSessionRevokeOthers, cfg.Logger)
	handlerSessionDelete := handlers.NewRestSessionDelete(serviceSessionDelete, cfg.Logger)

	handlerAuthorizationRequestValidate := handlers.NewRestAuthorizationRequestValidate(
		serviceAuthorizationRequestValidate, cfg.Logger,
	)
	handlerAuthorizationCodeCreate := handlers.NewRestAuthorizationCodeCreate(serviceAuthorizationCodeCreate, cfg.Logger)
	handlerTokenGrant := handlers.NewRestTokenGrant(serviceTokenGrant, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewMux(cfg.Rest, cfg.RestLogger, cfg.App.DowntimeStart)

	router.Route("/v2", func(api chi.Router) {
		api.Get("/ping", httpf.Ping)
		api.Get("/healthcheck", handlerHealth)

		withAuth(api, "waitlist:join").With(limit.Waitlist).Put("/waitlist", handlerWaitlistJoin.ServeHTTP)

		api.Route("/session", func(r chi.Router) {
			r.With(limit.Login).Put("/", handlerTokenCreate.ServeHTTP)
			r.With(limit.Anonymous).Put("/anon", handlerTokenCreateAnon.ServeHTTP)

			withSessionAuth(r).Get("/", handlerClaimsGet.ServeHTTP)
			r.Patch("/", handlerTokenRefresh.ServeHTTP)
			withSessionAuth(r, "session:revoke").Put("/revoke", handlerTokenRevoke.ServeHTTP)
		})

		// The account platform alone manages the devices of an account, so a client's token reaches none
		// of these routes.
		api.Route("/sessions", func(r chi.Router) {
			withSessions := func(permission string) chi.Router {
				return withAuth(r, permission).With(limit.Sessions, refuseEndedSessions)
			}

			withSessions("session:list").Get("/", handlerSessionList.ServeHTTP)
			withSessions("session:revoke").Put("/revoke-others", handlerSessionRevokeOthers.ServeHTTP)
			withSessions("session:revoke").Put("/{sessionID}/revoke", handlerSessionRevoke.ServeHTTP)
			withSessions("session:delete").Delete("/{sessionID}", handlerSessionDelete.ServeHTTP)
		})

		// The account platform checks an authorization request, then asks for a code for the person
		// signed in there. The client trades the code at the token endpoint.
		api.Route("/oauth", func(r chi.Router) {
			r.Get("/authorization-request", handlerAuthorizationRequestValidate.ServeHTTP)
			withAuth(r, "authorizationCode:create").With(limit.AuthorizationCode).
				Post("/authorization-code", handlerAuthorizationCodeCreate.ServeHTTP)
			r.With(limit.OAuthToken).Post("/token", handlerTokenGrant.ServeHTTP)
		})

		api.Route("/credentials", func(r chi.Router) {
			withAuth(r, "credentials:get").Get("/", handlerCredentialsGet.ServeHTTP)
			withAuth(r, "credentials:exist").Head("/", handlerCredentialsExist.ServeHTTP)
			withAuth(r, "credentials:list").Get("/all", handlerCredentialsList.ServeHTTP)
			withSessionAuth(r, "credentials:self:get").Get("/self", handlerCredentialsGetSelf.ServeHTTP)

			withAuth(r, "credentials:create").With(limit.SignUp).Put("/", handlerCredentialsCreate.ServeHTTP)
			withAuth(r, "credentials:email:patch").With(limit.ShortCode).
				Patch("/email", handlerCredentialsUpdateEmail.ServeHTTP)
			withAuth(r, "credentials:password:patch").With(limit.Password).
				Patch("/password", handlerCredentialsUpdatePassword.ServeHTTP)
			withAuth(r, "credentials:password:reset").With(limit.ShortCode).
				Put("/password", handlerCredentialsResetPassword.ServeHTTP)
			withAuth(r, "credentials:role:patch").
				Patch("/role", handlerCredentialsUpdateRole.ServeHTTP)
			withAuth(r, "credentials:locationConsent:patch").With(limit.Sessions, refuseEndedSessions).
				Patch("/location-consent", handlerCredentialsUpdateLocationConsent.ServeHTTP)
		})

		api.Route("/short-code", func(r chi.Router) {
			withAuth(r, "shortCode:register").With(limit.Mailbox).
				Put("/register", handlerShortCodeCreateRegister.ServeHTTP)
			withAuth(r, "shortCode:email:update").With(limit.Mail).
				Put("/update-email", handlerShortCodeCreateEmailUpdate.ServeHTTP)
			withAuth(r, "shortCode:email:update").With(limit.Sessions, refuseEndedSessions).
				Delete("/update-email", handlerShortCodeCancelEmailUpdate.ServeHTTP)
			withAuth(r, "shortCode:password:reset").With(limit.Mail).
				Put("/update-password", handlerShortCodeCreatePasswordReset.ServeHTTP)
		})
	})

	return withServiceToken(router), nil
}

// serve runs the shared HTTP lifecycle, then drains accepted mail inside the same shutdown budget.
func serve(
	ctx context.Context,
	stop context.CancelFunc,
	rest httpf.ServerConfig,
	handler http.Handler,
	mailDelivery *core.MailDelivery,
) error {
	shutdownTimeout := rest.Timeouts.Shutdown
	shutdownStarted := make(chan time.Time, 1)

	go func() {
		<-ctx.Done()
		mailDelivery.Close()

		shutdownStarted <- time.Now()
	}()

	log.Println("Starting REST server on port " + strconv.Itoa(rest.Port))

	serveErr := httpf.ListenAndServe(ctx, rest, handler)

	// Stop also cancels the process context when serving ends because of a bind or runtime error.
	// The recorded deadline lets the HTTP drain and accepted mail share one operator budget.
	stop()

	shutdownCtx, cancel := context.WithDeadline(
		context.WithoutCancel(ctx),
		(<-shutdownStarted).Add(shutdownTimeout),
	)

	log.Println("Draining in-flight emails...")

	drainErr := mailDelivery.Wait(shutdownCtx)

	cancel()

	if drainErr != nil {
		// Logged while the process is already stopping. Failed deliveries or mail that missed
		// the budget are worth reporting and the shutdown still completes.
		log.Println("Some emails failed or were still in flight at shutdown: " + drainErr.Error())
	}

	return serveErr
}
