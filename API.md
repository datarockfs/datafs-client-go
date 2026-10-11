# DataFS public HTTP contract

This is the surface the public SDKs and API clients call. Phase 2 of `docs/PLAN.md` implements it. Field limits match the design doc.

Base URL is the install, for example `http://localhost:8080`. No trailing slash.

JSON errors are `{"error":"<code>","message":"<short>"}`. `error` is stable. `message` is not localized.

## Tracking

`POST /api/event`

No API key. Auth is the site host allowlist. Body is JSON with `Content-Type: text/plain;charset=UTF-8`. Success is **202** with an empty body.

```json
{"n":"pageview","u":"https://example.com/pricing","d":"example.com","r":"https://news.ycombinator.com/","s":"hn","m":"social","c":"launch","id":"optional-key"}
```

| Field | Meaning |
| --- | --- |
| `n` | Event name. Pageviews send `pageview`. |
| `u` | Page URL. Query and hash are stripped before send. |
| `d` | Site host. |
| `r` | Referrer, query and hash stripped. Omit when empty. |
| `s`, `m`, `c` | `utm_source`, `utm_medium`, `utm_campaign` only. Each cut at 64 bytes. |
| `id` | Optional idempotency key, `[A-Za-z0-9_-]{1,64}`. |
| `p` | Props. Custom events only. At most 10 keys. Key length ≤ 32. Value is a string, length ≤ 64, and must not be an email. |

`utm_content` and `utm_term` are not sent. Names match `^[A-Za-z0-9_:\-]{1,64}$`. `pageview` and `engagement` are reserved for the script and the SDK pageview helper. The SDK does not retry a failed pageview.

## Server ingest

`POST /api/v1/server-event`

`Authorization: Bearer <ingest token>`. `Content-Type: application/json`. Success is **202** `{"event_id":"..."}`.

```json
{
  "domain": "example.com",
  "name": "signup",
  "url": "https://example.com/welcome",
  "referrer": "https://example.com/pricing",
  "props": {"plan": "pro"},
  "ip": "203.0.113.4",
  "user_agent": "Mozilla/5.0",
  "visitor_key": "customer-anonymous-session",
  "timestamp": "2026-10-11T12:00:00Z",
  "idempotency_key": "order-123"
}
```

Omit fields you do not have. Do not send an empty string. The same name and prop rules apply. `timestamp` is ISO-8601 UTC.

## Stats

`Authorization: Bearer <stats api key>`.

`GET /api/v1/version` → `{"version":"0.1.0","commit":"abc"}`. No auth required. Clients still send the key when they have one.

`GET /api/v1/sites/{siteID}/timeseries?from=YYYY-MM-DD&to=YYYY-MM-DD&interval=day|hour`

`from` and `to` are inclusive civil dates in the site timezone.

```json
{
  "from": "2026-09-12",
  "to": "2026-10-11",
  "interval": "day",
  "stale": false,
  "rollup_freshness": "2026-10-11T12:00:30Z",
  "totals": {"visitors": 100, "pageviews": 240, "visits": 130, "bounce_rate": 0.41, "visit_duration_seconds": 38},
  "points": [{"t": "2026-10-11", "visitors": 10, "pageviews": 22, "visits": 12, "bounce_rate": 0.33, "visit_duration_seconds": 40}]
}
```

`GET /api/v1/sites/{siteID}/breakdown?from&to&dimension&limit`

`dimension` is one of `page`, `entry_page`, `exit_page`, `referrer`, `channel`, `country`, `region`, `browser`, `os`, `device`, `goal`, `utm_source`, `utm_medium`, `utm_campaign`.

`country` is sent only when `dimension=region`.

```json
{
  "from": "2026-09-12",
  "to": "2026-10-11",
  "dimension": "page",
  "rows": [{"value": "/pricing", "visitors": 10, "pageviews": 22, "entrances": 0, "exits": 0}]
}
```

`entry_page` fills `entrances`. `exit_page` fills `exits`. Other dimensions fill `pageviews`.

`GET /api/v1/sites/{siteID}/realtime` → `{"visitors": 3}`.

`POST /api/v1/sites/{siteID}/funnel`

```json
{"day":"2026-10-10","steps":[{"kind":"path_exact","value":"/pricing"},{"kind":"event","value":"signup"}]}
```

`kind` is `path_exact`, `path_prefix`, or `event`. One to three steps. `day` is one site-local day.

```json
{"day":"2026-10-10","steps":[{"kind":"path_exact","value":"/pricing","visitors":40},{"kind":"event","value":"signup","visitors":9}]}
```
