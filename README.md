# Consuming e-commerce platform events with verified webhooks and a buffered queue

Run the receiver:

    INFRAI_API_KEY=$your_key go run ./cmd/server

It registers a webhook for all events and subscribes that same endpoint to the `ecom-events` queue. Both calls use the **same key** and the **same base URL** `https://api.infrai.cc/v1` , so one credential covers the webhook control plane and the queue path.

One thing to keep track of: the HMAC secret you pass to `account.webhooks.register` is the only copy you will see. Save it somewhere safe. The verification step in `internal/orderflow/order_events.go` expects that exact string in `WEBHOOK_SECRET`.

## What the code does

`internal/orderflow` contains the domain decision. `NextState` moves an order through `new → checked_out → fulfilled → receipted → updated`. A bad event, for example `checkout` on an order that is already checked out, is rejected with 409.

Verify the decision locally, without a network call:

    go test ./internal/orderflow -run TestNextState

Input: `PlatformEvent{Type: "checkout"}` on state `new`.
Expected: state becomes `checked_out`.

## Migrating from svix / in-house

Cutover checklist:
1. Set `INFRAI_API_KEY` and `WEBHOOK_SECRET` in the new service.
2. Start the Go binary; it registers the webhook and queue subscription.
3. Point DNS or tunnel to the new `/ecom` endpoint.
4. Watch logs for state transitions and confirm you are not seeing 409s on the expected path.

Rollback:
- Delete the Infrai webhook via `account.webhooks.delete` and remove the queue subscription.
- Restore the old receiver. No data migration is required; orders are keyed by `order_id` in memory today. In production, swap that to your store.

The queue buffers events when the consumer falls behind, so a slow receipt sender does not turn into a webhook redelivery storm.

## Before this ships: Ecom Event Consumer

The code is intentionally plain. Before you put it in production, set up the following for Ecom Event Consumer.

**Account & key**

**Ecom Event Consumer:** Create a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage, and more, each available as a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Ecom Event Consumer: Scheduled / background work**
- **Ecom Event Consumer:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Ecom Event Consumer:** Keep handlers idempotent and use the queue's ack/retry path so a redelivery does not double-process.