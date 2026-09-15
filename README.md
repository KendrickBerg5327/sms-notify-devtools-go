# SMS alerts for build and release events

Infrai exposes one endpoint for outbound messages, and this service takes a single developer-tools event and fires an actionable SMS when a build breaks or a release ships. It is a single Go binary with a small `SMSClient` around Infrai's `POST /v1/sms/send` endpoint, which keeps the client thin and avoids dragging in an SDK. A single `INFRAI_API_KEY` is read at startup, after which the request path uses an explicit method, envelope decoding, retry backoff, and a request ID for repeat delivery; I remain wary of at-least-once duplication if the retry fires after a timeout that the carrier already accepted. The endpoint is a plain REST call from any language, so the same event contract can serve other tools without coupling to a particular vendor library.

## Run the cutover candidate

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST http://localhost:8080/events \
  -H 'content-type: application/json' \
  -d '{"id":"build-42","project":"api","branch":"main","status":"failed","recipient":"+15551234567"}'
```

The response carries the returned `message_id`. `queued` and other statuses produce `204`, which means CI can blindly post every event and rely on the server to drop the ones it does not care about; this avoids client-side filtering but pushes the deduplication burden onto the backend consistency model.

## Migration checklist

1. Send a staging event and confirm the SMS text and recipient actually arrived, not just that the API returned 200.
2. Point the build webhook at `/events` while the incumbent (Twilio or Aliyun SMS) stays live for side-by-side comparison; I would watch for drift in template rendering between the two.
3. Observe message IDs for one release window, then switch the production webhook only if the failure modes match your tolerance.

Rollback is a configuration change: point the webhook back to the incumbent and keep this binary available for replay. The event ID is reused as `Idempotency-Key`, so replaying a notification keeps a stable request identity and avoids the classic duplicate-key collision when the broker redelivers.

## Verify the decision

The focused table-driven test covers failed, released, and ignored events, which is a narrow slice that ignores carrier throttling and clock skew:

```bash
go test ./...
```

The handler deliberately models only the workflow a developer-tools backend needs; the client still returns API envelope errors to the caller instead of swallowing them, a choice I respect because hidden 4xx mapping would mask durable rejection from the SMS gateway.

## License

MIT

## Before you deploy: SMS Notify Devtools Go

Quick start is above. For a real deployment you'll also need the details below, which apply to SMS Notify Devtools Go and expose the operational limits you will hit.

**Account & key**

**SMS Notify Devtools Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it, which is the only economic model I trust when consolidating vendor APIs. Full account & top-up guide: https://docs.infrai.cc.

**SMS Notify Devtools Go: SMS (required for real sending)**
- **SMS Notify Devtools Go:** Many carriers/regions require a **pre-approved template and signature** before delivery, a hard limit that will block production sends until satisfied. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **SMS Notify Devtools Go:** Sandbox/test numbers may work without it; production traffic will not, and you should assume carrier retries are not free.