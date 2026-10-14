"""Apache-2.0 SDK for the LIMENORA Enterprise Agent Integration Gateway.

Exposes the public surface: gateway role/info, semantic version comparison, and
the FP-1 federated-presentation client contract (shared with the gateway so a
client binds the same target the gateway verifies).
"""

from __future__ import annotations

from dataclasses import dataclass
from enum import Enum

from .policy import (
    CONNECT_PATH,
    REVISION_PATH,
    CacheScope,
    ConnectDecision,
    ConnectRequest,
    ConnectResponse,
    InjectionRule,
    PolicyClient,
    PolicyRule,
)
from .network import (
    CARRIAGE_CONNECT_HEADER,
    CARRIAGE_DISABLED,
    CARRIAGE_REQUEST_HEADER,
    AgentTokenSource,
    GatewayRequest,
    GatewayResult,
    NoAgentToken,
    ProofCarrier,
    ProxyEgress,
    StaticTokenSource,
    metadata_denied,
)

#: Public SDK version. Tracks the core module tag.
VERSION = "2.4.0"


class Role(str, Enum):
    """A gateway deployment role."""

    EDGE = "edge"
    ENTERPRISE = "enterprise"
    CONTROL = "control"


@dataclass(frozen=True)
class GatewayInfo:
    """A stable summary of a running gateway instance."""

    version: str
    role: Role


def info(role: Role) -> GatewayInfo:
    """Return a stable :class:`GatewayInfo` for ``role``."""
    return GatewayInfo(version=VERSION, role=role)


def _split(version: str):
    v = version[1:] if version.startswith("v") else version
    v = v.split("+", 1)[0]
    if "-" in v:
        main, pre = v.split("-", 1)
        return main.split("."), pre.split(".")
    return v.split("."), []


def _cmp_segments(a, b) -> int:
    n = max(len(a), len(b))
    for i in range(n):
        x = a[i] if i < len(a) else ""
        y = b[i] if i < len(b) else ""
        if x == y:
            continue
        try:
            xn = int(x)
            yn = int(y)
            return -1 if xn < yn else 1
        except ValueError:
            return 1 if y == "" else -1
    return 0


def compare_versions(a: str, b: str) -> int:
    """Order two version strings semantically (returns -1, 0, or 1)."""
    am, ap = _split(a)
    bm, bp = _split(b)
    c = _cmp_segments(am, bm)
    if c != 0:
        return c
    if not ap and not bp:
        return 0
    if not ap:
        return 1
    if not bp:
        return -1
    return _cmp_segments(ap, bp)


class Carriage(str, Enum):
    """Where a resource proof is carried; visible to both client and gateway."""

    REQUEST_HEADER = "request_header"
    CONNECT_HEADER = "connect_header"
    DISABLED = "disabled"


class TunnelPolicy(str, Enum):
    """How a raw, non-intercepted tunnel carrying agent traffic is treated."""

    DENY = "deny"
    AUTHORITY_BINDING = "authority_binding"
    PASSTHROUGH_NON_AGENT = "passthrough_non_agent"


DEFAULT_RESOURCE_PROOF_HEADER = "DPoP"
DEFAULT_RESOURCE_PROOF_SCHEME = "DPoP"
DEFAULT_INTROSPECTION_PROOF_HEADER = "DPoP-Introspection"
DEFAULT_INTROSPECTION_PROOF_SCHEME = "DPoP"

__all__ = [
    "VERSION",
    "Role",
    "GatewayInfo",
    "info",
    "compare_versions",
    "Carriage",
    "TunnelPolicy",
    "DEFAULT_RESOURCE_PROOF_HEADER",
    "DEFAULT_RESOURCE_PROOF_SCHEME",
    "DEFAULT_INTROSPECTION_PROOF_HEADER",
    "DEFAULT_INTROSPECTION_PROOF_SCHEME",
    "CONNECT_PATH",
    "REVISION_PATH",
    "ConnectRequest",
    "ConnectResponse",
    "ConnectDecision",
    "PolicyRule",
    "InjectionRule",
    "CacheScope",
    "PolicyClient",
    "CARRIAGE_REQUEST_HEADER",
    "CARRIAGE_CONNECT_HEADER",
    "CARRIAGE_DISABLED",
    "NoAgentToken",
    "metadata_denied",
    "AgentTokenSource",
    "StaticTokenSource",
    "ProofCarrier",
    "GatewayRequest",
    "GatewayResult",
    "ProxyEgress",
]
