/**
 * Apache-2.0 SDK for the LIMENORA Enterprise Agent Integration Gateway.
 *
 * Exposes the public surface: gateway role/info, semantic version comparison,
 * and the FP-1 federated-presentation client contract (shared with the gateway
 * so a client binds the same target the gateway verifies).
 */

/** Public SDK version. Tracks the core module tag. */
export const VERSION = "2.4.0";

/** A gateway deployment role. */
export type Role = "edge" | "enterprise" | "control";

/** A stable summary of a running gateway instance. */
export interface GatewayInfo {
  version: string;
  role: Role;
}

/** Return a stable {@link GatewayInfo} for `role`. */
export function info(role: Role): GatewayInfo {
  return { version: VERSION, role };
}

function split(v: string): [string[], string[]] {
  let s = v.startsWith("v") ? v.slice(1) : v;
  s = s.split("+")[0];
  const dash = s.indexOf("-");
  if (dash >= 0) {
    return [s.slice(0, dash).split("."), s.slice(dash + 1).split(".")];
  }
  return [s.split("."), []];
}

function cmpSegments(a: string[], b: string[]): number {
  const n = Math.max(a.length, b.length);
  for (let i = 0; i < n; i++) {
    const x = a[i] ?? "";
    const y = b[i] ?? "";
    if (x === y) continue;
    const xn = Number(x);
    const yn = Number(y);
    if (!Number.isNaN(xn) && !Number.isNaN(yn)) return xn < yn ? -1 : 1;
    if (!Number.isNaN(xn)) return 1;
    if (!Number.isNaN(yn)) return -1;
    return x < y ? -1 : 1;
  }
  return 0;
}

/** Order two version strings semantically (returns -1, 0, or 1). */
export function compareVersions(a: string, b: string): number {
  const [am, ap] = split(a);
  const [bm, bp] = split(b);
  const c = cmpSegments(am, bm);
  if (c !== 0) return c;
  if (ap.length === 0 && bp.length === 0) return 0;
  if (ap.length === 0) return 1;
  if (bp.length === 0) return -1;
  return cmpSegments(ap, bp);
}

/** Where a resource proof is carried and which target its `htu` binds. */
export type Carriage = "request_header" | "connect_header" | "disabled";

/** How a raw, non-intercepted tunnel carrying agent traffic is treated. */
export type TunnelPolicy = "deny" | "authority_binding" | "passthrough_non_agent";

/** FP-1 proof carriage header/scheme defaults. */
export const DEFAULT_RESOURCE_PROOF_HEADER = "DPoP";
export const DEFAULT_RESOURCE_PROOF_SCHEME = "DPoP";
export const DEFAULT_INTROSPECTION_PROOF_HEADER = "DPoP-Introspection";
export const DEFAULT_INTROSPECTION_PROOF_SCHEME = "DPoP";

export * from "./policy";
export * from "./network";

