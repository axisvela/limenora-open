//! LIMENORA proxy-egress client (Apache-2.0).
//!
//! Routes outbound requests through a LIMENORA forward proxy; the gateway
//! credential is presented via `Proxy-Authorization` (Bearer), never the
//! destination `Authorization`. Cloud-metadata destinations are denied and the
//! response body is bounded. HTTPS uses the proxy CONNECT tunnel (reqwest).

use std::time::Duration;

use reqwest::blocking::Client;
use reqwest::header::{HeaderMap, HeaderName, HeaderValue};
use reqwest::Proxy;

pub const DEFAULT_MAX_RESPONSE_BYTES: usize = 10 << 20; // 10 MiB

const METADATA_HOSTS: &[&str] = &["metadata.google.internal", "metadata.azure.internal"];
const METADATA_IPS: &[&str] = &["169.254.169.254", "168.63.129.16", "fe80::a9fe:a9fe", "fd00:ec2::254"];

fn is_metadata_ip(host: &str) -> bool {
    if METADATA_IPS.contains(&host) {
        return true;
    }
    // AWS ECS metadata range 169.254.170.0/23.
    if let Some(rest) = host.strip_prefix("169.254.17") {
        return rest.starts_with('0') || rest.starts_with('1');
    }
    false
}

/// Reports whether `host` names a cloud metadata endpoint.
pub fn metadata_denied(host: &str) -> bool {
    let h = host.trim().to_ascii_lowercase();
    if h.is_empty() {
        return false;
    }
    if METADATA_HOSTS.contains(&h.as_str()) || is_metadata_ip(&h) {
        return true;
    }
    // Strip an optional port.
    if let Some((bare, _)) = h.rsplit_once(':') {
        if METADATA_HOSTS.contains(&bare) || is_metadata_ip(bare) {
            return true;
        }
    }
    false
}

/// Yields a bearer token for the proxy (e.g. an AEGIVELA workload assertion).
pub trait TokenSource: Send + Sync {
    fn token(&self) -> Result<String, String>;
}

/// A fixed-token source.
pub struct StaticTokenSource(pub String);

impl TokenSource for StaticTokenSource {
    fn token(&self) -> Result<String, String> {
        Ok(self.0.clone())
    }
}

/// An outbound request to route through the proxy.
#[derive(Default)]
pub struct GatewayRequest {
    pub url: String,
    pub method: String,
    pub headers: Vec<(String, String)>,
    pub body: Option<Vec<u8>>,
    pub trace_id: String,
}

/// The proxied response.
pub struct GatewayResult {
    pub status: u16,
    pub body: Vec<u8>,
    pub headers: Vec<(String, String)>,
}

/// A proxy-egress error.
#[derive(Debug)]
pub enum EgressError {
    Denied(String),
    NoToken,
    Transport(String),
    TooLarge(usize),
}

impl std::fmt::Display for EgressError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            EgressError::Denied(h) => write!(f, "destination denied: cloud metadata endpoint {h}"),
            EgressError::NoToken => write!(f, "no agent token"),
            EgressError::Transport(m) => write!(f, "transport: {m}"),
            EgressError::TooLarge(n) => write!(f, "response exceeds {n} bytes"),
        }
    }
}

impl std::error::Error for EgressError {}

/// The proxy-egress client.
pub struct ProxyEgress {
    proxy_base: String,
    tokens: Box<dyn TokenSource>,
    max: usize,
    timeout: Duration,
}

impl ProxyEgress {
    /// Creates an egress client for `proxy_base` (absolute http(s)).
    pub fn new(proxy_base: &str, tokens: Box<dyn TokenSource>) -> Result<Self, EgressError> {
        let base = proxy_base.trim_end_matches('/').to_string();
        if !(base.starts_with("http://") || base.starts_with("https://")) {
            return Err(EgressError::Transport(format!("proxy URL must be absolute http(s): {proxy_base}")));
        }
        Ok(Self { proxy_base: base, tokens, max: DEFAULT_MAX_RESPONSE_BYTES, timeout: Duration::from_secs(30) })
    }

    /// Bounds the response body.
    pub fn with_max_response_bytes(mut self, n: usize) -> Self {
        if n > 0 {
            self.max = n;
        }
        self
    }

    /// Sends `req` through the proxy.
    pub fn do_request(&self, req: &GatewayRequest) -> Result<GatewayResult, EgressError> {
        // Deny cloud metadata destinations before any egress.
        if let Ok(parsed) = reqwest::Url::parse(&req.url) {
            if let Some(host) = parsed.host_str() {
                if metadata_denied(host) {
                    return Err(EgressError::Denied(host.to_string()));
                }
            }
        } else {
            return Err(EgressError::Transport(format!("invalid URL: {}", req.url)));
        }

        let token = self.tokens.token().map_err(|_| EgressError::NoToken)?;
        if token.is_empty() {
            return Err(EgressError::NoToken);
        }
        let auth = HeaderValue::from_str(&format!("Bearer {token}"))
            .map_err(|e| EgressError::Transport(e.to_string()))?;

        // Build a client per call so the (rotating) proxy credential is applied.
        let proxy = Proxy::all(&self.proxy_base)
            .map_err(|e| EgressError::Transport(e.to_string()))?
            .custom_http_auth(auth);
        let client = Client::builder()
            .proxy(proxy)
            .timeout(self.timeout)
            .build()
            .map_err(|e| EgressError::Transport(e.to_string()))?;

        let method = if req.method.is_empty() {
            reqwest::Method::GET
        } else {
            reqwest::Method::from_bytes(req.method.as_bytes()).map_err(|e| EgressError::Transport(e.to_string()))?
        };
        let mut rb = client.request(method, &req.url);

        let mut headers = HeaderMap::new();
        for (k, v) in &req.headers {
            // Proxy-Authorization is a proxy credential, never a destination
            // header; drop any caller-supplied value (reqwest sets its own).
            if k.eq_ignore_ascii_case("proxy-authorization") {
                continue;
            }
            if let (Ok(name), Ok(value)) = (HeaderName::from_bytes(k.as_bytes()), HeaderValue::from_str(v)) {
                headers.insert(name, value);
            }
        }
        if !req.trace_id.is_empty() {
            if let Ok(v) = HeaderValue::from_str(&req.trace_id) {
                headers.insert(HeaderName::from_static("x-trace-id"), v);
            }
        }
        rb = rb.headers(headers);
        if let Some(body) = &req.body {
            rb = rb.body(body.clone());
        }

        let resp = rb.send().map_err(|e| EgressError::Transport(e.to_string()))?;
        let status = resp.status().as_u16();
        let resp_headers: Vec<(String, String)> = resp
            .headers()
            .iter()
            .map(|(k, v)| (k.as_str().to_string(), v.to_str().unwrap_or("").to_string()))
            .collect();
        let bytes = resp.bytes().map_err(|e| EgressError::Transport(e.to_string()))?;
        if bytes.len() > self.max {
            return Err(EgressError::TooLarge(self.max));
        }
        Ok(GatewayResult { status, body: bytes.to_vec(), headers: resp_headers })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn metadata_denied_rules() {
        assert!(metadata_denied("169.254.169.254"));
        assert!(metadata_denied("metadata.google.internal"));
        assert!(!metadata_denied("example.com"));
    }

    #[test]
    fn new_requires_absolute_proxy() {
        assert!(ProxyEgress::new("not-a-url", Box::new(StaticTokenSource("t".into()))).is_err());
    }
}
