"""LIMENORA policy-plane client (Apache-2.0).

CONNECT policy decision and content revision, from the shared public contract
(contracts/api/policy-connect-*.schema.json). Stdlib-only (urllib).
"""

from __future__ import annotations

import json
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import asdict, dataclass, field
from typing import Any, Dict, List, Mapping, Optional

CONNECT_PATH = "/v1/policy/connect"
REVISION_PATH = "/v1/policy/revision"


@dataclass
class ConnectRequest:
    agent_token: str
    host: str
    port: str = ""
    scheme: str = ""
    method: str = ""
    path: str = ""
    headers: Dict[str, str] = field(default_factory=dict)
    connection_id: str = ""
    gateway_role: str = ""
    gateway_version: str = ""
    agent_ref: str = ""
    agent_id: str = ""
    agent_class: str = ""
    identity_type: str = ""
    project_id: str = ""
    tenant_id: str = ""
    organization_id: str = ""
    audience: str = ""

    def to_wire(self) -> Dict[str, Any]:
        return {k: v for k, v in asdict(self).items() if v not in ("", {}, None)}


@dataclass
class PolicyRule:
    id: str = ""
    name: str = ""
    host_pattern: str = ""
    path_pattern: str = ""
    method: str = ""
    action: Dict[str, Any] = field(default_factory=dict)
    conditions: Dict[str, Any] = field(default_factory=dict)


@dataclass
class InjectionRule:
    source: str = ""
    location: str = ""
    name: str = ""
    id: str = ""
    value: str = ""
    host_pattern: str = ""
    path_pattern: str = ""
    metadata: Dict[str, str] = field(default_factory=dict)
    oauth_provider: str = ""
    credential_binding_ref: str = ""


@dataclass
class CacheScope:
    tenant_id: str = ""
    project_id: str = ""
    organization_id: str = ""
    agent_id: str = ""
    hosts: List[str] = field(default_factory=list)
    provider_ids: List[str] = field(default_factory=list)


@dataclass
class ConnectDecision:
    intercept: bool = False
    intercept_reason: str = ""
    tenant_id: str = ""
    project_id: str = ""
    organization_id: str = ""
    agent_id: str = ""
    agent_name: str = ""
    agent_identifier: str = ""
    agent_ref: str = ""
    agent_class: str = ""
    identity_type: str = ""
    access_restricted: bool = False
    policy_mode: str = ""
    policy_rules: List[PolicyRule] = field(default_factory=list)
    injection_rules: List[InjectionRule] = field(default_factory=list)
    provider_ids: List[str] = field(default_factory=list)
    app_connection_ids: List[str] = field(default_factory=list)
    body_transform: str = ""
    cache_scope: Optional[CacheScope] = None
    cache_revision: str = ""

    @classmethod
    def from_wire(cls, data: Mapping[str, Any]) -> "ConnectDecision":
        d = dict(data or {})
        d["policy_rules"] = [PolicyRule(**r) for r in d.get("policy_rules", [])]
        d["injection_rules"] = [InjectionRule(**r) for r in d.get("injection_rules", [])]
        if isinstance(d.get("cache_scope"), dict):
            d["cache_scope"] = CacheScope(**d["cache_scope"])
        # Drop unknown keys so server extensions do not break decoding.
        known = {f for f in cls.__dataclass_fields__}
        return cls(**{k: v for k, v in d.items() if k in known})


@dataclass
class ConnectResponse:
    decision: ConnectDecision
    revision: str = ""


class PolicyClient:
    def __init__(self, base_url: str, timeout: float = 10.0):
        base = (base_url or "").rstrip("/")
        if not base:
            raise ValueError("base URL is required")
        self.base_url = base
        self.timeout = timeout

    def _request(self, method: str, path: str, body: Optional[bytes]) -> Any:
        req = urllib.request.Request(self.base_url + path, data=body, method=method)
        if body is not None:
            req.add_header("Content-Type", "application/json")
        req.add_header("Accept", "application/json")
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            detail = ""
            try:
                detail = json.loads(e.read().decode("utf-8")).get("error", "")
            except Exception:
                pass
            raise RuntimeError(f"policy request failed: {detail or e.code}") from e

    def resolve_connect(self, request: ConnectRequest) -> ConnectResponse:
        body = json.dumps(request.to_wire()).encode("utf-8")
        out = self._request("POST", CONNECT_PATH, body)
        return ConnectResponse(
            decision=ConnectDecision.from_wire(out.get("decision", {})),
            revision=out.get("revision", ""),
        )

    def revision(self, agent_token: str) -> str:
        path = REVISION_PATH + "?agent_token=" + urllib.parse.quote(agent_token, safe="")
        out = self._request("GET", path, None)
        return out.get("revision", "")


__all__ = [
    "CONNECT_PATH",
    "REVISION_PATH",
    "ConnectRequest",
    "ConnectResponse",
    "ConnectDecision",
    "PolicyRule",
    "InjectionRule",
    "CacheScope",
    "PolicyClient",
]
