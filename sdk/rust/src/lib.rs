//! Apache-2.0 SDK for the LIMENORA Enterprise Agent Integration Gateway.
//!
//! Exposes the public surface: gateway role/info, semantic version comparison,
//! and the FP-1 federated-presentation client contract (shared with the
//! gateway so a client binds the same target the gateway verifies).

/// Public SDK version. It tracks the core module tag.
pub const VERSION: &str = "2.4.0";

/// A gateway deployment role.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Role {
    Edge,
    Enterprise,
    Control,
}

impl Role {
    pub fn as_str(self) -> &'static str {
        match self {
            Role::Edge => "edge",
            Role::Enterprise => "enterprise",
            Role::Control => "control",
        }
    }
}

/// A stable summary of a running gateway instance.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GatewayInfo {
    pub version: String,
    pub role: Role,
}

/// Returns a stable [`GatewayInfo`] for `role`.
pub fn info(role: Role) -> GatewayInfo {
    GatewayInfo {
        version: VERSION.to_string(),
        role,
    }
}

/// Orders two version strings semantically (leading `v` ignored; numeric
/// segments compare numerically; pre-release sorts below release).
pub fn compare_versions(a: &str, b: &str) -> i32 {
    let (am, ap) = split(a);
    let (bm, bp) = split(b);
    let c = cmp_segments(&am, &bm);
    if c != 0 {
        return c;
    }
    match (ap.is_empty(), bp.is_empty()) {
        (true, true) => 0,
        (true, false) => 1,
        (false, true) => -1,
        (false, false) => cmp_segments(&ap, &bp),
    }
}

fn split(v: &str) -> (Vec<String>, Vec<String>) {
    let v = v.strip_prefix('v').unwrap_or(v);
    let v = v.split('+').next().unwrap_or(v);
    match v.split_once('-') {
        Some((m, p)) => (
            m.split('.').map(String::from).collect(),
            p.split('.').map(String::from).collect(),
        ),
        None => (v.split('.').map(String::from).collect(), Vec::new()),
    }
}

fn cmp_segments(a: &[String], b: &[String]) -> i32 {
    let n = a.len().max(b.len());
    for i in 0..n {
        let as_ = a.get(i).map(String::as_str).unwrap_or("");
        let bs = b.get(i).map(String::as_str).unwrap_or("");
        if as_ == bs {
            continue;
        }
        let an = as_.parse::<i64>();
        let bn = bs.parse::<i64>();
        match (an, bn) {
            (Ok(x), Ok(y)) => return if x < y { -1 } else { 1 },
            (Ok(_), Err(_)) => return 1,
            (Err(_), Ok(_)) => return -1,
            (Err(_), Err(_)) => return if as_ < bs { -1 } else { 1 },
        }
    }
    0
}

/// Where a resource proof is carried and which target its `htu` binds.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Carriage {
    RequestHeader,
    ConnectHeader,
    Disabled,
}

impl Carriage {
    pub fn as_str(self) -> &'static str {
        match self {
            Carriage::RequestHeader => "request_header",
            Carriage::ConnectHeader => "connect_header",
            Carriage::Disabled => "disabled",
        }
    }
    pub fn parse(s: &str) -> Option<Self> {
        match s {
            "request_header" => Some(Carriage::RequestHeader),
            "connect_header" => Some(Carriage::ConnectHeader),
            "disabled" => Some(Carriage::Disabled),
            _ => None,
        }
    }
}

/// How a raw, non-intercepted tunnel carrying selected agent traffic is treated.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TunnelPolicy {
    Deny,
    AuthorityBinding,
    PassthroughNonAgent,
}

impl TunnelPolicy {
    pub fn as_str(self) -> &'static str {
        match self {
            TunnelPolicy::Deny => "deny",
            TunnelPolicy::AuthorityBinding => "authority_binding",
            TunnelPolicy::PassthroughNonAgent => "passthrough_non_agent",
        }
    }
    pub fn parse(s: &str) -> Option<Self> {
        match s {
            "deny" => Some(TunnelPolicy::Deny),
            "authority_binding" => Some(TunnelPolicy::AuthorityBinding),
            "passthrough_non_agent" => Some(TunnelPolicy::PassthroughNonAgent),
            _ => None,
        }
    }
}

/// FP-1 proof carriage header/scheme defaults.
pub const DEFAULT_RESOURCE_PROOF_HEADER: &str = "DPoP";
pub const DEFAULT_RESOURCE_PROOF_SCHEME: &str = "DPoP";
pub const DEFAULT_INTROSPECTION_PROOF_HEADER: &str = "DPoP-Introspection";
pub const DEFAULT_INTROSPECTION_PROOF_SCHEME: &str = "DPoP";

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn info_reports_version_and_role() {
        let i = info(Role::Enterprise);
        assert_eq!(i.version, VERSION);
        assert_eq!(i.role.as_str(), "enterprise");
    }

    #[test]
    fn compare_versions_semantics() {
        assert_eq!(compare_versions("1.10", "1.9"), 1);
        assert_eq!(compare_versions("v2.3.0", "2.3.0"), 0);
        assert_eq!(compare_versions("2.0.0-rc1", "2.0.0"), -1);
        assert_eq!(compare_versions("1.0.0", "1.0.0"), 0);
    }

    #[test]
    fn carriage_and_tunnel_roundtrip() {
        assert_eq!(Carriage::parse("request_header"), Some(Carriage::RequestHeader));
        assert_eq!(Carriage::parse("bogus"), None);
        assert_eq!(TunnelPolicy::parse("deny"), Some(TunnelPolicy::Deny));
        assert_eq!(TunnelPolicy::parse("bogus"), None);
    }
}
