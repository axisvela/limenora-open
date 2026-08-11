# Webhook Receiver Example

This example shows how an external system delivers signed webhooks through the
LIMENORA ingress boundary.

Planned flow (Wave 2):

- External partner sends a signed webhook to the LIMENORA ingress.
- LIMENORA validates signature, timestamp/nonce, and dedupes by idempotency key.
- LIMENORA routes the event to the target Capability / workflow endpoint.

Concrete runnable code will be added when the webhook ingress milestone lands
in Wave 2.
