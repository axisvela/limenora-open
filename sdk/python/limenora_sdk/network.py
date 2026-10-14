"""LIMENORA proxy-egress client (Apache-2.0).

Ports the Go `network.ProxyEgress`: routes outbound requests through a LIMENORA
forward proxy (absolute-form for http, CONNECT tunnel + TLS for https), presents
the gateway credential via Proxy-Authorization (never the destination
Authorization), denies cloud-metadata destinations, binds the response size, and
fails closed on a missing credential. Stdlib-only.

Redirects are not auto-followed (returned to the caller) so destination
credentials never cross an origin boundary.
"""

from __future__ import annotations

import ipaddress
import ssl
from dataclasses import dataclass, field
from typing import Dict, List, Mapping, Optional, Protocol
from urllib.parse import urlparse

import http.client

DEFAULT_MAX_RESPONSE_BYTES = 10 << 20  # 10 MiB

CARRIAGE_REQUEST_HEADER = "request_header"
CARRIAGE_CONNECT_HEADER = "connect_header"
CARRIAGE_DISABLED = "disabled"

_METADATA_HOSTS = {"metadata.google.internal", "metadata.azure.internal"}
_METADATA_IPS = {
    ipaddress.ip_address("169.254.169.254"),
    ipaddress.ip_address("168.63.129.16"),
    ipaddress.ip_address("fe80::a9fe:a9fe"),
    ipaddress.ip_address("fd00:ec2::254"),
}
_METADATA_PREFIXES = [ipaddress.ip_network("169.254.170.0/23")]


class NoAgentToken(Exception):
    """Raised when no gateway credential is available (fail closed)."""


def _in_metadata_ips(ip: ipaddress._BaseAddress) -> bool:
    if ip in _METADATA_IPS:
        return True
    return any(ip in net for net in _METADATA_PREFIXES)


def metadata_denied(host: str) -> bool:
    h = (host or "").strip().lower()
    if not h:
        return False
    try:
        return _in_metadata_ips(ipaddress.ip_address(h))
    except ValueError:
        pass
    # Strip port / brackets then retry as a bare host or IP.
    try:
        bare = urlparse("//" + h).hostname or h
    except ValueError:
        bare = h
    if bare in _METADATA_HOSTS:
        return True
    try:
        return _in_metadata_ips(ipaddress.ip_address(bare))
    except ValueError:
        return False


def request_denied_host(url: str) -> str:
    try:
        host = urlparse(url).hostname
    except ValueError:
        return ""
    if host and metadata_denied(host):
        return host
    return ""


class AgentTokenSource(Protocol):
    def token(self) -> str: ...


class StaticTokenSource:
    def __init__(self, token: str) -> None:
        self._token = token

    def token(self) -> str:
        return self._token


class ProofCarrier(Protocol):
    def proof_headers(self, method: str, target: str, token: str) -> Mapping[str, str]: ...


@dataclass
class GatewayRequest:
    url: str
    method: str = "GET"
    headers: Dict[str, str] = field(default_factory=dict)
    body: Optional[bytes] = None
    trace_id: str = ""


@dataclass
class GatewayResult:
    status: int
    body: bytes
    headers: Dict[str, str] = field(default_factory=dict)


class ProxyEgress:
    def __init__(
        self,
        proxy_base: str,
        tokens: AgentTokenSource,
        max_response_bytes: int = DEFAULT_MAX_RESPONSE_BYTES,
        carriage: str = CARRIAGE_REQUEST_HEADER,
        proofs: Optional[ProofCarrier] = None,
        tls_context: Optional[ssl.SSLContext] = None,
        timeout: float = 30.0,
    ) -> None:
        if tokens is None:
            raise ValueError("proxy egress: nil token source")
        u = urlparse(proxy_base or "")
        if not u.hostname or u.scheme not in ("http", "https"):
            raise ValueError(f"proxy egress: proxy URL must be absolute http(s): {proxy_base!r}")
        self._proxy = u
        self._tokens = tokens
        self._max = max_response_bytes if max_response_bytes > 0 else DEFAULT_MAX_RESPONSE_BYTES
        self._carriage = carriage or CARRIAGE_REQUEST_HEADER
        self._proofs = proofs
        self._tls = tls_context or ssl.create_default_context()
        self._timeout = timeout

    def _proxy_conn(self) -> http.client.HTTPConnection:
        host = self._proxy.hostname
        port = self._proxy.port or (443 if self._proxy.scheme == "https" else 80)
        if self._proxy.scheme == "https":
            return http.client.HTTPSConnection(host, port, timeout=self._timeout, context=self._tls)
        return http.client.HTTPConnection(host, port, timeout=self._timeout)

    def do(self, req: GatewayRequest) -> GatewayResult:
        denied = request_denied_host(req.url)
        if denied:
            raise ValueError(f"network: destination denied: cloud metadata endpoint {denied!r}")

        token = self._tokens.token()
        if not token:
            raise NoAgentToken("network: proxy egress: no agent token")

        target = urlparse(req.url)
        if not target.hostname:
            raise ValueError(f"network: proxy egress: invalid target URL {req.url!r}")

        headers = dict(req.headers)
        headers.pop("Proxy-Authorization", None)
        if self._proofs is not None and self._carriage == CARRIAGE_REQUEST_HEADER:
            headers.update(self._proofs.proof_headers(req.method, req.url, token))
        if req.trace_id:
            headers["X-Trace-ID"] = req.trace_id

        if target.scheme == "https":
            return self._do_connect(req, target, headers, token)
        return self._do_absolute(req, headers, token)

    def _do_absolute(self, req: GatewayRequest, headers: Dict[str, str], token: str) -> GatewayResult:
        headers["Proxy-Authorization"] = "Bearer " + token
        conn = self._proxy_conn()
        try:
            conn.request(req.method, req.url, body=req.body, headers=headers)
            resp = conn.getresponse()
            return self._read_response(resp)
        finally:
            conn.close()

    def _do_connect(self, req: GatewayRequest, target, headers: Dict[str, str], token: str) -> GatewayResult:
        port = target.port or 443
        hostport = f"{target.hostname}:{port}"
        connect_headers = {"Proxy-Authorization": "Bearer " + token}
        if self._proofs is not None and self._carriage == CARRIAGE_CONNECT_HEADER:
            connect_headers.update(self._proofs.proof_headers("CONNECT", f"https://{hostport}", token))

        proxy = self._proxy_conn()
        try:
            proxy.request("CONNECT", hostport, headers=connect_headers)
            tunnel = proxy.getresponse()
            if tunnel.status != 200:
                raise ValueError(f"network: proxy egress: CONNECT failed: {tunnel.status} {tunnel.reason}")
            raw_sock = proxy.sock
            tls_sock = self._tls.wrap_socket(raw_sock, server_hostname=target.hostname)

            origin = http.client.HTTPConnection(target.hostname, port, timeout=self._timeout)
            origin.sock = tls_sock  # send the request over the established tunnel
            path = target.path or "/"
            if target.query:
                path += "?" + target.query
            origin.request(req.method, path, body=req.body, headers=headers)
            resp = origin.getresponse()
            return self._read_response(resp)
        finally:
            proxy.close()

    def _read_response(self, resp: http.client.HTTPResponse) -> GatewayResult:
        raw = resp.read(self._max + 1)
        if len(raw) > self._max:
            raise ValueError(f"network: proxy egress: response exceeds {self._max} bytes")
        headers = {k: v for k, v in resp.getheaders()}
        return GatewayResult(status=resp.status, body=raw, headers=headers)


__all__ = [
    "DEFAULT_MAX_RESPONSE_BYTES",
    "CARRIAGE_REQUEST_HEADER",
    "CARRIAGE_CONNECT_HEADER",
    "CARRIAGE_DISABLED",
    "NoAgentToken",
    "metadata_denied",
    "request_denied_host",
    "AgentTokenSource",
    "StaticTokenSource",
    "ProofCarrier",
    "GatewayRequest",
    "GatewayResult",
    "ProxyEgress",
]
