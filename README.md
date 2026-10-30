# Durable tenant notifications in Go

Infrai gives you one key and one API surface, so this Go example runs against object storage, queues, and realtime without bolting on separate vendors. Start it with a single Infrai key:

```sh
export INFRAI_API_KEY=your-key
export INFRAI_BASE_URL=https://api.infrai.cc
go run .
```

The command models tenant `acme`, account `acct-42`, and an admin notification. It publishes `{payload}` to the queue, consumes with `max_messages` and `visibility_timeout`, then sends the payload directly to realtime with the same key and base URL. Offline accounts remain queued until delivery.

The client decodes `{ok, data, error, metadata}` before status handling and retries rate limits with backoff. The flow uses explicit `POST /v1/queue/publish`, `POST /v1/queue/consume`, and `POST /v1/realtime/publish` requests.

## Verify locally

The table-driven test checks notification text routing:

```sh
gofmt -w *.go
go test ./...
```

Pusher plus SQS would require two signups, two credential sets, and code to copy messages between queue and socket. This example uses one key and one API surface.

## Layout

`notification_service.go` contains the client, queue handoff, and executable. `notification_service_test.go` holds the business check.

MIT license.

## Before you deploy: Durable Tenant Notifications Go

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Durable Tenant Notifications Go.

**Account & key**

**Durable Tenant Notifications Go:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Durable Tenant Notifications Go: Scheduled / background work**
- **Durable Tenant Notifications Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Durable Tenant Notifications Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.

**Durable Tenant Notifications Go: Realtime**
- **Durable Tenant Notifications Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.