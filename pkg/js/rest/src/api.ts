import { decodeHttpResponse, handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

import type { ZodType } from "zod";

async function decodeRawHttpResponse<T>(response: Response): Promise<T> {
  return await response.json();
}

/** Status of a single health-check dependency reported by the `/healthcheck` endpoint. */
export type HealthDependency = {
  /** Whether the dependency is reachable. */
  status: "up" | "down";
  /** Failure detail, present only when the dependency is `down`. */
  err?: string;
};

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
export class AuthenticationApi {
  private readonly _baseUrl: string;

  constructor(baseUrl: string) {
    this._baseUrl = baseUrl;
  }

  /**
   * Sends a request to the given path and discards the response body.
   * Throws if the server returns a non-2xx status.
   */
  async fetchVoid(input: string, init?: RequestInit): Promise<void> {
    await fetch(`${this._baseUrl}${input}`, init).then(handleHttpResponse);
  }

  /**
   * Sends a request to the given path and deserializes the JSON response body as `T`.
   * When a Zod validator is supplied, the body is parsed through it; otherwise it is returned
   * as-is. Throws if the server returns a non-2xx status or the body fails validation.
   */
  async fetch<T>(input: string, validator?: ZodType<T>, init?: RequestInit): Promise<T> {
    return await fetch(`${this._baseUrl}${input}`, init)
      .then(handleHttpResponse)
      .then(validator ? decodeHttpResponse(validator) : decodeRawHttpResponse<T>);
  }

  /** Checks that the server is reachable. Throws on any non-2xx response. */
  async ping(): Promise<void> {
    await this.fetchVoid("/ping", { method: "GET" });
  }

  /**
   * Returns the health status of every service dependency, keyed by dependency name.
   * The endpoint always responds 200; a degraded dependency shows as a `down` entry,
   * so inspect each entry's `status` field to detect one.
   */
  async health(): Promise<Record<string, HealthDependency>> {
    return await this.fetch("/healthcheck", undefined, { method: "GET" });
  }
}
