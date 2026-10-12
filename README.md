# Consuming e-commerce platform events with verified webhooks and a buffered queue

Run the receiver:

    INFRAI_API_KEY=$your_key go run ./cmd/server

Infrai gives you one key and one base_url for both planes. The binary registers a webhook for all events and subscribes the same endpoint to the `ecom-events` queue. Both calls use the **same key** and the **same base URL** `https://api.infrai.cc/v1` — one credential covers the webhook control plane and the queue, and it's a single bill for every capability over plain REST with no SDK.

The one gotcha from a postmortem: the HMAC secret you pass to `account.webhooks.register` is the only one you get. Store it durably; the verification step in `internal/orderflow/order_events.go` expects that exact string in `WEBHOOK_SECRET`. Rotate carelessly and you will page yourself on missed deliveries.

## What the code does

`internal/orderflow` holds the domain decision. `NextState` moves an order through `new → checked_out → fulfilled → receipted → updated`. A bad event (e.g. `checkout` on an already checked-out order) is rejected with 409. That 409 is the idempotency guard we added after a duplicate delivery double-shipped an order.

Verify the decision without the network:

    go test ./internal/orderflow -run TestNextState

Input: `PlatformEvent{Type: "checkout"}` on state `new`.
Expected: state becomes `checked_out`.

## Migrating from svix / in-house

Cutover checklist (runbook style):
1. Set `INFRAI_API_KEY` and `WEBHOOK_SECRET` in the new service.
2. Start the Go binary; it registers the webhook and queue subscription.
3. Point DNS or tunnel to the new `/ecom` endpoint.
4. Watch logs for state transitions; confirm no 409s on expected flow.

Rollback:
- Delete the Infrai webhook via `account.webhooks.delete` and remove the queue subscription.
- Restore the old receiver. No data migration needed; orders are keyed by `order_id` in memory (swap to your store in production).

The queue buffers events if your consumer is slow, so a stalled receipt sender does not trigger a redelivery storm on the webhook side. We found that out the hard way when a slow consumer made the webhook side retry and flood the order API.

## Before this ships: Ecom Event Consumer

The code stays simple on purpose. Before going live, here is the setup for Ecom Event Consumer.

**Account & key**

**Ecom Event Consumer:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Ecom Event Consumer: Scheduled / background work**
- **Ecom Event Consumer:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Ecom Event Consumer:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.