/**
 * LIMENORA proxy-egress client (Apache-2.0), Node-only.
 *
 * Ports the Go `network.ProxyEgress`: absolute-form http through the proxy,
 * CONNECT + TLS for https, Proxy-Authorization gateway credential (never the
 * destination Authorization), cloud-metadata deny, bounded body, fail-closed on
 * a missing credential. Uses node:http/https/tls (not the fetch API).
 *
 * Redirects are not auto-followed (returned to the caller).
 */

import http from "node:http";
import https from "node:https";
import tls from "node:tls";
import { URL } from "node:url";

export const DEFAULT_MAX_RESPONSE_BYTES = 10 << 20; // 10 MiB
export const CARRIAGE_REQUEST_HEADER = "request_header";
export const CARRIAGE_CONNECT_HEADER = "connect_header";
export const CARRIAGE_DISABLED = "disabled";

const METADATA_HOSTS = new Set(["metadata.google.internal", "metadata.azure.internal"]);
const METADATA_IPS = new Set(["169.254.169.254", "168.63.129.16", "fe80::a9fe:a9fe", "fd00:ec2::254", "169.254.170.5"]);
// 169.254.170.0/23 covers AWS ECS metadata.
function isMetadataIp(host: string): boolean {
  if (METADATA_IPS.has(host)) return true;
  const m = /^169\.254\.(170|171)\.\d{1,3}$/.exec(host);
  return m !== null;
}

export function metadataDenied(host: string): boolean {
  const h = (host ?? "").trim().toLowerCase();
  if (!h) return false;
  if (METADATA_HOSTS.has(h) || isMetadataIp(h)) return true;
  try {
    const bare = new URL("//" + h).hostname;
    return METADATA_HOSTS.has(bare) || isMetadataIp(bare);
  } catch {
    return false;
  }
}

export function requestDeniedHost(rawUrl: string): string {
  try {
    const host = new URL(rawUrl).hostname;
    return host && metadataDenied(host) ? host : "";
  } catch {
    return "";
  }
}

export class NoAgentToken extends Error {}

export interface AgentTokenSource {
  token(): string | Promise<string>;
}

export class StaticTokenSource implements AgentTokenSource {
  constructor(private readonly value: string) {}
  token(): string {
    return this.value;
  }
}

export interface ProofCarrier {
  proofHeaders(method: string, target: string, token: string): Record<string, string>;
}

export interface GatewayRequest {
  url: string;
  method?: string;
  headers?: Record<string, string>;
  body?: Uint8Array;
  traceId?: string;
}

export interface GatewayResult {
  status: number;
  body: Uint8Array;
  headers: Record<string, string>;
}

export interface ProxyEgressOptions {
  maxResponseBytes?: number;
  carriage?: string;
  proofs?: ProofCarrier;
  timeoutMs?: number;
  tls?: tls.ConnectionOptions;
}

export class ProxyEgress {
  private readonly proxy: URL;
  private readonly tokens: AgentTokenSource;
  private readonly max: number;
  private readonly carriage: string;
  private readonly proofs?: ProofCarrier;
  private readonly timeoutMs: number;
  private readonly tlsOpts: tls.ConnectionOptions;

  constructor(proxyBase: string, tokens: AgentTokenSource, opts: ProxyEgressOptions = {}) {
    if (!tokens) throw new Error("proxy egress: nil token source");
    const u = new URL(proxyBase);
    if (!u.hostname || (u.protocol !== "http:" && u.protocol !== "https:")) {
      throw new Error(`proxy egress: proxy URL must be absolute http(s): ${proxyBase}`);
    }
    this.proxy = u;
    this.tokens = tokens;
    this.max = opts.maxResponseBytes && opts.maxResponseBytes > 0 ? opts.maxResponseBytes : DEFAULT_MAX_RESPONSE_BYTES;
    this.carriage = opts.carriage ?? CARRIAGE_REQUEST_HEADER;
    this.proofs = opts.proofs;
    this.timeoutMs = opts.timeoutMs ?? 30000;
    this.tlsOpts = opts.tls ?? {};
  }

  async do(req: GatewayRequest): Promise<GatewayResult> {
    const denied = requestDeniedHost(req.url);
    if (denied) throw new Error(`network: destination denied: cloud metadata endpoint '${denied}'`);

    const token = await this.tokens.token();
    if (!token) throw new NoAgentToken("network: proxy egress: no agent token");

    const target = new URL(req.url);
    const headers: Record<string, string> = { ...(req.headers ?? {}) };
    delete headers["Proxy-Authorization"];
    if (this.proofs && this.carriage === CARRIAGE_REQUEST_HEADER) {
      Object.assign(headers, this.proofs.proofHeaders(req.method ?? "GET", req.url, token));
    }
    if (req.traceId) headers["X-Trace-ID"] = req.traceId;

    return target.protocol === "https:"
      ? this.doConnect(req, target, headers, token)
      : this.doAbsolute(req, headers, token);
  }

  private proxyPort(): number {
    if (this.proxy.port) return Number(this.proxy.port);
    return this.proxy.protocol === "https:" ? 443 : 80;
  }

  private doAbsolute(req: GatewayRequest, headers: Record<string, string>, token: string): Promise<GatewayResult> {
    headers["Proxy-Authorization"] = "Bearer " + token;
    const mod = this.proxy.protocol === "https:" ? https : http;
    const options: https.RequestOptions = {
      host: this.proxy.hostname,
      port: this.proxyPort(),
      method: req.method ?? "GET",
      path: req.url, // absolute-form
      headers,
      timeout: this.timeoutMs,
      ...(this.proxy.protocol === "https:" ? this.tlsOpts : {}),
    };
    return new Promise<GatewayResult>((resolve, reject) => {
      const r = mod.request(options, (res) => this.collect(res, resolve, reject));
      r.on("error", reject);
      if (req.body) r.write(req.body);
      r.end();
    });
  }

  private doConnect(req: GatewayRequest, target: URL, headers: Record<string, string>, token: string): Promise<GatewayResult> {
    const port = target.port ? Number(target.port) : 443;
    const hostport = `${target.hostname}:${port}`;
    const connectHeaders: Record<string, string> = { "Proxy-Authorization": "Bearer " + token };
    if (this.proofs && this.carriage === CARRIAGE_CONNECT_HEADER) {
      Object.assign(connectHeaders, this.proofs.proofHeaders("CONNECT", `https://${hostport}`, token));
    }
    return new Promise<GatewayResult>((resolve, reject) => {
      const connectReq = http.request({
        host: this.proxy.hostname,
        port: this.proxyPort(),
        method: "CONNECT",
        path: hostport,
        headers: connectHeaders,
        timeout: this.timeoutMs,
      });
      connectReq.on("connect", (res, socket) => {
        if (res.statusCode !== 200) {
          socket.destroy();
          reject(new Error(`network: proxy egress: CONNECT failed: ${res.statusCode}`));
          return;
        }
        const tlsSocket = tls.connect({ socket, servername: target.hostname, ...this.tlsOpts }, () => {
          const path = (target.pathname || "/") + (target.search || "");
          const r = https.request(
            {
              createConnection: () => tlsSocket,
              method: req.method ?? "GET",
              path,
              headers: { Host: target.host, ...headers },
              timeout: this.timeoutMs,
            },
            (tres) => this.collect(tres, resolve, reject),
          );
          r.on("error", reject);
          if (req.body) r.write(req.body);
          r.end();
        });
        tlsSocket.on("error", reject);
      });
      connectReq.on("error", reject);
      connectReq.end();
    });
  }

  private collect(
    res: http.IncomingMessage,
    resolve: (r: GatewayResult) => void,
    reject: (e: Error) => void,
  ): void {
    const chunks: Buffer[] = [];
    let size = 0;
    res.on("data", (chunk: Buffer) => {
      size += chunk.length;
      if (size > this.max) {
        res.destroy();
        reject(new Error(`network: proxy egress: response exceeds ${this.max} bytes`));
        return;
      }
      chunks.push(chunk);
    });
    res.on("end", () => {
      const headers: Record<string, string> = {};
      for (const [k, v] of Object.entries(res.headers)) {
        if (typeof v === "string") headers[k] = v;
        else if (Array.isArray(v)) headers[k] = v.join(", ");
      }
      resolve({ status: res.statusCode ?? 0, body: new Uint8Array(Buffer.concat(chunks)), headers });
    });
    res.on("error", reject);
  }
}
