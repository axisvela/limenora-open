//! LIMENORA policy-plane client (CONNECT decision + revision).
//!
//! Consumes the shared public contract (contracts/api/policy-connect-*).

use serde::{Deserialize, Serialize};
use std::time::Duration;

pub const CONNECT_PATH: &str = "/v1/policy/connect";
pub const REVISION_PATH: &str = "/v1/policy/revision";

/// A CONNECT policy decision request.
#[derive(Debug, Clone, Default, Serialize)]
pub struct ConnectRequest {
    pub agent_token: String,
    pub host: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub port: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scheme: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub method: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub path: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub connection_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gateway_role: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gateway_version: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub agent_ref: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub agent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub agent_class: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub identity_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub project_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tenant_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub audience: Option<String>,
}

/// A resolved CONNECT policy decision (rule/injection details kept as raw JSON).
#[derive(Debug, Clone, Default, Deserialize)]
pub struct ConnectDecision {
    #[serde(default)]
    pub intercept: bool,
    #[serde(default)]
    pub intercept_reason: Option<String>,
    #[serde(default)]
    pub tenant_id: Option<String>,
    #[serde(default)]
    pub project_id: Option<String>,
    #[serde(default)]
    pub organization_id: Option<String>,
    #[serde(default)]
    pub agent_id: Option<String>,
    #[serde(default)]
    pub agent_name: Option<String>,
    #[serde(default)]
    pub agent_identifier: Option<String>,
    #[serde(default)]
    pub agent_ref: Option<String>,
    #[serde(default)]
    pub agent_class: Option<String>,
    #[serde(default)]
    pub identity_type: Option<String>,
    #[serde(default)]
    pub access_restricted: bool,
    #[serde(default)]
    pub policy_mode: Option<String>,
    #[serde(default)]
    pub policy_rules: Option<serde_json::Value>,
    #[serde(default)]
    pub injection_rules: Option<serde_json::Value>,
    #[serde(default)]
    pub provider_ids: Option<Vec<String>>,
    #[serde(default)]
    pub app_connection_ids: Option<Vec<String>>,
    #[serde(default)]
    pub body_transform: Option<String>,
    #[serde(default)]
    pub cache_scope: Option<serde_json::Value>,
    #[serde(default)]
    pub cache_revision: Option<String>,
}

/// The policy decision plus the source content revision.
#[derive(Debug, Clone, Deserialize)]
pub struct ConnectResponse {
    pub decision: ConnectDecision,
    #[serde(default)]
    pub revision: Option<String>,
}

/// A policy-client error.
#[derive(Debug)]
pub enum PolicyError {
    Transport(String),
    Status(u16, String),
    Decode(String),
}

impl std::fmt::Display for PolicyError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            PolicyError::Transport(m) => write!(f, "policy transport: {m}"),
            PolicyError::Status(s, b) => write!(f, "policy status {s}: {b}"),
            PolicyError::Decode(m) => write!(f, "policy decode: {m}"),
        }
    }
}

impl std::error::Error for PolicyError {}

/// The LIMENORA policy-plane client.
pub struct PolicyClient {
    base_url: String,
    http: reqwest::blocking::Client,
}

impl PolicyClient {
    /// Creates a policy client for `base_url` (required).
    pub fn new(base_url: &str) -> Result<Self, PolicyError> {
        let base = base_url.trim_end_matches('/').to_string();
        if base.is_empty() {
            return Err(PolicyError::Transport("base URL is required".into()));
        }
        let http = reqwest::blocking::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .map_err(|e| PolicyError::Transport(e.to_string()))?;
        Ok(Self { base_url: base, http })
    }

    /// Requests a CONNECT policy decision.
    pub fn resolve_connect(&self, req: &ConnectRequest) -> Result<ConnectResponse, PolicyError> {
        let resp = self
            .http
            .post(format!("{}{}", self.base_url, CONNECT_PATH))
            .json(req)
            .send()
            .map_err(|e| PolicyError::Transport(e.to_string()))?;
        let status = resp.status();
        if !status.is_success() {
            return Err(PolicyError::Status(status.as_u16(), resp.text().unwrap_or_default()));
        }
        resp.json::<ConnectResponse>()
            .map_err(|e| PolicyError::Decode(e.to_string()))
    }

    /// Fetches the content revision for the agent token.
    pub fn revision(&self, agent_token: &str) -> Result<String, PolicyError> {
        let resp = self
            .http
            .get(format!("{}{}", self.base_url, REVISION_PATH))
            .query(&[("agent_token", agent_token)])
            .send()
            .map_err(|e| PolicyError::Transport(e.to_string()))?;
        let status = resp.status();
        if !status.is_success() {
            return Err(PolicyError::Status(status.as_u16(), resp.text().unwrap_or_default()));
        }
        let v: serde_json::Value = resp
            .json()
            .map_err(|e| PolicyError::Decode(e.to_string()))?;
        Ok(v.get("revision").and_then(|r| r.as_str()).unwrap_or("").to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn connect_request_serializes() {
        let req = ConnectRequest {
            agent_token: "at".into(),
            host: "example.com".into(),
            ..Default::default()
        };
        let json = serde_json::to_string(&req).unwrap();
        assert!(json.contains("\"agent_token\":\"at\""));
        assert!(json.contains("\"host\":\"example.com\""));
        // Optional fields are omitted when None.
        assert!(!json.contains("port"));
    }

    #[test]
    fn new_requires_base_url() {
        assert!(PolicyClient::new("").is_err());
    }

    #[test]
    fn decision_decodes() {
        let d: ConnectDecision =
            serde_json::from_str(r#"{"intercept":true,"intercept_reason":"policy_rule"}"#).unwrap();
        assert!(d.intercept);
        assert_eq!(d.intercept_reason.as_deref(), Some("policy_rule"));
    }
}
