/**
 * LIMENORA policy-plane client (Apache-2.0).
 *
 * CONNECT policy decision and content revision, from the shared public contract
 * (contracts/api/policy-connect-*.schema.json). Uses global fetch (Node >= 18).
 */

export const CONNECT_PATH = "/v1/policy/connect";
export const REVISION_PATH = "/v1/policy/revision";

export interface ConnectRequest {
  agent_token: string;
  host: string;
  port?: string;
  scheme?: string;
  method?: string;
  path?: string;
  headers?: Record<string, string>;
  connection_id?: string;
  gateway_role?: string;
  gateway_version?: string;
  agent_ref?: string;
  agent_id?: string;
  agent_class?: string;
  identity_type?: string;
  project_id?: string;
  tenant_id?: string;
  organization_id?: string;
  audience?: string;
}

export interface PolicyRule {
  id: string;
  name: string;
  host_pattern: string;
  path_pattern: string;
  method?: string;
  action: { type: string; rule_id?: string; max_requests?: number; window_secs?: number };
  conditions?: Record<string, unknown>;
}

export interface InjectionRule {
  id?: string;
  source: string;
  location: string;
  name: string;
  value?: string;
  host_pattern?: string;
  path_pattern?: string;
  metadata?: Record<string, string>;
  oauth_provider?: string;
  credential_binding_ref?: string;
}

export interface CacheScope {
  tenant_id?: string;
  project_id?: string;
  organization_id?: string;
  agent_id?: string;
  hosts?: string[];
  provider_ids?: string[];
}

export interface ConnectDecision {
  intercept: boolean;
  intercept_reason?: string;
  tenant_id?: string;
  project_id?: string;
  organization_id?: string;
  agent_id?: string;
  agent_name?: string;
  agent_identifier?: string;
  agent_ref?: string;
  agent_class?: string;
  identity_type?: string;
  access_restricted?: boolean;
  policy_mode?: string;
  policy_rules?: PolicyRule[];
  injection_rules?: InjectionRule[];
  provider_ids?: string[];
  app_connection_ids?: string[];
  body_transform?: string;
  cache_scope?: CacheScope;
  cache_revision?: string;
}

export interface ConnectResponse {
  decision: ConnectDecision;
  revision?: string;
}

function stripEmpty<T extends Record<string, unknown>>(obj: T): Partial<T> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v === "" || v === undefined || v === null) continue;
    if (Array.isArray(v) && v.length === 0) continue;
    if (typeof v === "object" && !Array.isArray(v) && Object.keys(v).length === 0) continue;
    out[k] = v;
  }
  return out as Partial<T>;
}

export class PolicyClient {
  private baseURL: string;
  private timeoutMs: number;

  constructor(baseURL: string, timeoutMs = 10000) {
    const base = (baseURL ?? "").replace(/\/+$/, "");
    if (!base) throw new Error("base URL is required");
    this.baseURL = base;
    this.timeoutMs = timeoutMs;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);
    try {
      const resp = await fetch(this.baseURL + path, {
        method,
        headers: body === undefined ? { Accept: "application/json" } : { "Content-Type": "application/json", Accept: "application/json" },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: controller.signal,
      });
      if (!resp.ok) {
        let detail = resp.statusText;
        try {
          const err = (await resp.json()) as { error?: string };
          if (err?.error) detail = err.error;
        } catch {
          /* ignore */
        }
        throw new Error(`policy request failed: ${detail}`);
      }
      return (await resp.json()) as T;
    } finally {
      clearTimeout(timer);
    }
  }

  async resolveConnect(request: ConnectRequest): Promise<ConnectResponse> {
    const wire = stripEmpty(request as unknown as Record<string, unknown>);
    const out = await this.request<ConnectResponse>("POST", CONNECT_PATH, wire);
    return { decision: out.decision, revision: out.revision };
  }

  async revision(agentToken: string): Promise<string> {
    const path = `${REVISION_PATH}?agent_token=${encodeURIComponent(agentToken)}`;
    const out = await this.request<{ revision?: string }>("GET", path);
    return out.revision ?? "";
  }
}
