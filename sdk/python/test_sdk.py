"""Tests for the LIMENORA Python SDK (policy + network)."""

import http.server
import threading
import unittest

import limenora_sdk as sdk


class _FakeProxy(http.server.BaseHTTPRequestHandler):
    seen = []

    def do_GET(self):  # noqa: N802
        _FakeProxy.seen.append((self.command, self.path, dict(self.headers)))
        body = b"ok"
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args):  # silence
        pass


class NetworkTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = http.server.HTTPServer(("127.0.0.1", 0), _FakeProxy)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.proxy_url = f"http://127.0.0.1:{cls.server.server_address[1]}"

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def test_metadata_denied(self):
        self.assertTrue(sdk.metadata_denied("169.254.169.254"))
        self.assertTrue(sdk.metadata_denied("metadata.google.internal"))
        self.assertFalse(sdk.metadata_denied("example.com"))

    def test_metadata_target_fails_closed(self):
        e = sdk.ProxyEgress(self.proxy_url, sdk.StaticTokenSource("tok"))
        with self.assertRaises(ValueError):
            e.do(sdk.GatewayRequest(url="http://169.254.169.254/latest/meta-data"))

    def test_missing_token_fails_closed(self):
        e = sdk.ProxyEgress(self.proxy_url, sdk.StaticTokenSource(""))
        with self.assertRaises(sdk.NoAgentToken):
            e.do(sdk.GatewayRequest(url="http://example.com/"))

    def test_absolute_form_through_proxy(self):
        _FakeProxy.seen.clear()
        e = sdk.ProxyEgress(self.proxy_url, sdk.StaticTokenSource("tok-1"))
        res = e.do(sdk.GatewayRequest(url="http://example.com/path", trace_id="t-1"))
        self.assertEqual(res.status, 200)
        self.assertEqual(res.body, b"ok")
        method, path, headers = _FakeProxy.seen[-1]
        self.assertEqual(path, "http://example.com/path")
        self.assertEqual(headers.get("Proxy-Authorization"), "Bearer tok-1")
        self.assertEqual(headers.get("X-Trace-ID"), "t-1")


class PolicyTests(unittest.TestCase):
    def test_client_requires_base_url(self):
        with self.assertRaises(ValueError):
            sdk.PolicyClient("")


if __name__ == "__main__":
    unittest.main()
