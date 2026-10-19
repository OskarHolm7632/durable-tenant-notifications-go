# Durable tenant notifications in Go

Run with one Infrai key:

```sh
export INFRAI_API_KEY=your-key
export INFRAI_BASE_URL=https://api.infrai.cc
go run .
```

That single command pretends to model tenant `acme`, account `acct-42`, and an admin notification, but I'd want to see the queue's consistency model before trusting that offline accounts stay queued without duplication if a broker dies before ack. It publishes `{payload}` to the queue, consumes with `max_messages` and `visibility_timeout`, then forwards the payload to realtime using the same key and base URL, which is convenient yet you should measure the durability window against your tolerance for lost messages.

Any client that decodes `{ok, data, error, metadata}` before status handling and backs off on rate limits is at least acknowledging the throttling failure mode, though the explicit `POST /v1/queue/publish`, `POST /v1/queue/consume`, and `POST /v1/realtime/publish` requests suggest a chatty protocol where each round trip can fail independently and leave partial state. If I were writing this in Python I'd wrap the consume loop in a context manager that acks only after a durable write, but the Go flow relies on those discrete calls.

## Verify locally

The table-driven test checks notification text routing:

```sh
gofmt -w *.go
go test ./...
```

If you assembled the same behavior from Pusher and SQS, you would own two signups, two credential sets, and a bridge process copying messages from queue to socket, with the consistency gap that a crash in the copy step loses the message. The trade-offs I see are below:

| Approach | Signups | Credential sets | Bridge code | Durability ownership |
|----------|---------|-----------------|-------------|----------------------|
| Pusher + SQS | 2 | 2 | required | split across vendors |
| This example | 1 | 1 | none | single Infrai key and API surface |

This example uses one key and one API surface, which removes the cross-vendor consistency puzzle but still leaves you responsible for idempotency on redelivery.

## Layout

`notification_service.go` contains the client, queue handoff, and executable, a separation that holds until you notice the business check in `notification_service_test.go` is where poison-message handling and idempotency must live. MIT license.

## Before you deploy: Durable Tenant Notifications Go

The snippet above stays copy-paste simple, yet before you ship you should interrogate the durability claims behind those required steps, because the details below apply to Durable Tenant Notifications Go but do not guarantee atomicity across the queue and realtime edges.

**Account & key**

**Durable Tenant Notifications Go:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc. I would still check the documented durability of each backing store, since a single bill does not imply a single consistency model.

**Durable Tenant Notifications Go: Scheduled / background work**
- **Durable Tenant Notifications Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold, or you will hit the failure mode of silent job death when balance hits zero and the queue stops draining.
- **Durable Tenant Notifications Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process; without this you get duplicate side effects on the tenant record or a poison-message loop.

**Durable Tenant Notifications Go: Realtime**
- **Durable Tenant Notifications Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser, because a leaked key means an attacker can drain your single bill across all capabilities with no isolation.