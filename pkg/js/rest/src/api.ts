import { type HealthDependency, RestClient } from "@a-novel-kit/nodelib-browser/http";

export type { HealthDependency };

/**
 * HTTP client for the authentication REST API.
 *
 * The authentication service owns user credentials, sessions, and roles. Its REST API issues
 * and refreshes the access/refresh token pairs that authenticate every other service, manages
 * credential records, and drives the short-code flows that confirm sensitive actions by email.
 *
 * The endpoint helpers in this package (`tokenCreate`, `claimsGet`, `credentialsGet`, ...) each
 * take an instance of this client as their first argument.
 */
export class AuthenticationApi extends RestClient {
  /** Checks that the server is reachable. Throws on any non-2xx response. */
  async ping(): Promise<void> {
    await this.fetchVoid("/v2/ping");
  }

  /**
   * Returns the dependency health report when every probe succeeds.
   * Throws on HTTP 503 when any dependency is unhealthy.
   */
  async health(): Promise<Record<string, HealthDependency>> {
    return await this.fetch("/v2/healthcheck");
  }
}
