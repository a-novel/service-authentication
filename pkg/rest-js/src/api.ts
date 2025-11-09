import { decodeHttpResponse, handleHttpResponse } from "@a-novel/nodelib-browser/http";

import type { ZodType } from "zod";

async function decodeRawHttpResponse<T>(response: Response): Promise<T> {
  return await response.json();
}

export type HealthDependency = {
  status: "up" | "down";
  err?: string;
};

export class AuthenticationApi {
  private readonly _baseUrl: string;

  constructor(baseUrl: string) {
    this._baseUrl = baseUrl;
  }

  async fetchVoid(input: string, init?: RequestInit): Promise<void> {
    await fetch(`${this._baseUrl}${input}`, init).then(handleHttpResponse);
  }

  async fetch<T>(input: string, validator?: ZodType<T>, init?: RequestInit): Promise<T> {
    return await fetch(`${this._baseUrl}${input}`, init)
      .then(handleHttpResponse)
      .then(validator ? decodeHttpResponse(validator) : decodeRawHttpResponse<T>);
  }

  async ping(): Promise<void> {
    await this.fetchVoid("/ping", { method: "GET" });
  }

  async health(): Promise<Record<string, HealthDependency>> {
    return await this.fetch("/healthcheck", undefined, { method: "GET" });
  }
}
