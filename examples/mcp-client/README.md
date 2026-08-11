# MCP Client Example

This example shows how an agent connects to a remote MCP server through the
LIMENORA gateway.

Planned flow (Wave 1):

- Agent tool call targets a remote MCP server.
- LIMENORA resolves the MCP Route / ProtocolBinding.
- LIMENORA enforces AEGIVELA policy and injects credentials.
- The remote MCP server receives the governed request.

Concrete runnable code will be added when the remote MCP gateway milestone
lands in Wave 1.
