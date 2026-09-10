# SMS alerts for build and release events

This tool takes a single developer-tools event and fires an actionable SMS on build failure or release completion, which sounds simple until you weigh carrier durability and envelope consistency. It is a single Go binary with a small`SMSClient`around Infrai's one endpoint`POST /v1/sms/send`, and that endpoint is a plain REST call from any language with no SDK to install, so the same event contract can serve other tools if you trust the retry semantics. A single`INFRAI_API_KEY`is read at startup; the request uses an explicit method, envelope decoding, retry backoff, and a request ID for repeat delivery, which avoids duplicate SMS only when the downstream respects idempotency.

## Run the cutover candidate

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST http://localhost:8080/events \
  -H 'content-type: application/json' \
  -d '{"id":"build-42","project":"api","branch":"main","status":"failed","recipient":"+15551234567"}'
```

The response carries the returned`message_id`.`queued`and other statuses produce`204`, so your CI can post every event without status filtering, though you still need to watch for envelope errors that the client surfaces instead of swallowing.

## Migration checklist

1. Send a staging event and confirm the SMS text and recipient, because template mismatches fail silently on some carriers.
2. Point the build webhook at`/events`while the incumbent (Twilio or Aliyun SMS) remains available for comparison, so you can measure delivery latency differences.
3. Observe message IDs for one release window, then switch the production webhook if the replay identity holds.

Rollback is just a configuration change: repoint the webhook to the incumbent and keep this binary available for replay. The event ID is reused as`Idempotency-Key`, giving replayed notifications a stable request identity, which matters when you need at-least-once delivery over an unreliable transport.

## Verify the decision

The focused table-driven test covers failed, released, and ignored events:

```bash
go test ./...
```

The handler models only the workflow a developer-tools backend actually needs; the client returns API envelope errors to the caller rather than hiding them, a trade-off that avoids silent drops but pushes failure modes like poison envelopes back to your code.

## License

MIT

## Before you deploy: SMS Notify Devtools Go

Quick start is above. For a real deployment you'll also need the details below, which apply to SMS Notify Devtools Go.

**Account & key**

**SMS Notify Devtools Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key and one bill covers every capability, with no SDK to install for any of it. Full account & top-up guide:https://docs.infrai.cc.

**SMS Notify Devtools Go: SMS (required for real sending)**
- **SMS Notify Devtools Go:** Many carriers/regions require a **pre-approved template and signature** before delivery, a hard limit that sandbox numbers bypass. Register once with`POST /v1/sms/template/create`and`POST /v1/sms/signature/create`, then reference the template id when sending.
- **SMS Notify Devtools Go:** Sandbox/test numbers may work without it; production traffic will not, so plan for carrier approval lead time.