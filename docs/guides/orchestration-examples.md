---
title: Orchestration Engine — Examples
date: 2026-07-23
---

# 🎛 Orchestration Engine — Examples

Worked examples for the orchestration engine described in
[docs/concepts/orchestration-engine.md](../concepts/orchestration-engine.md).
All requests go through the tenant dispatcher (`/t/{org_handle}/...`) and
require a bearer token — see the main [README](../../README.md) for how to
obtain one. Examples below use `carbon.super` as the org handle and
`$TOKEN` for the token; `$CDS` is the base URL (e.g. `https://localhost:8900`).

Required scopes: `internal_cds_event_create` / `internal_cds_event_view` for
`/events`, `internal_cds_orchestration_rule_{view,create,update,delete}` for
`/orchestration-rules`, and `internal_cds_notification_template_{view,create,update,delete}`
for `/notification-templates`.

---

## 1. app.callback — trigger an in-app banner/prompt

The application registers its own endpoint in the rule's config; CDS POSTs
the event + profile to it whenever the rule matches, and the application
decides what to render.

```bash
curl -sk -X POST "$CDS/t/carbon.super/cds/api/v1/orchestration-rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "cart-abandonment-banner",
    "trigger": { "event_type": "track", "event_name": "add_to_cart" },
    "conditions": [
      { "field": "event.properties.value", "operator": "gt", "value": "40" }
    ],
    "actions": [
      {
        "type": "app.callback",
        "config": {
          "endpoint_url": "https://app.example.com/cds/callback",
          "secret": "a-shared-hmac-secret",
          "payload": {
            "banner": "high_value_cart_reminder",
            "profile_id": "{{event.profile_id}}",
            "cart_value": "{{event.properties.value}}"
          }
        }
      }
    ],
    "priority": 10,
    "is_active": true
  }'
```

The application verifies the `X-CDS-Signature: sha256=<hmac>` header (HMAC-SHA256
of the raw request body, keyed with `secret`) before acting on the callback.

---

## 2. notify.email — send an email

Requires `notifications.smtp.host` to be set in `deployment.yaml` (empty by
default — the action fails with a clear error until configured).

```bash
curl -sk -X POST "$CDS/t/carbon.super/cds/api/v1/orchestration-rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "welcome-email",
    "trigger": { "event_type": "identify", "event_name": "signup_completed" },
    "actions": [
      {
        "type": "notify.email",
        "config": {
          "to": "{{profile.identity_attributes.emailaddress}}",
          "subject": "Welcome!",
          "body": "Hi {{profile.traits.first_name}}, thanks for signing up."
        }
      }
    ],
    "priority": 10,
    "is_active": true
  }'
```

---

## 2b. Reusable templates instead of inlining subject/body

Rather than copy-pasting `subject`/`body` into every rule, define the content
once and reference it by `template_id`. `subject`/`body` (or `message` for
SMS) on the action, if also set, override the template's fields — you don't
have to choose all-or-nothing.

```bash
curl -sk -X POST "$CDS/t/carbon.super/cds/api/v1/notification-templates" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "email",
    "name": "welcome-email",
    "subject": "Welcome!",
    "body": "Hi {{profile.traits.first_name}}, thanks for signing up."
  }'
```

Response includes the assigned `template_id`; reference it from the action instead of inlining content:

```json
{
  "type": "notify.email",
  "config": {
    "to": "{{profile.identity_attributes.emailaddress}}",
    "template_id": "1e2d3c4b-..."
  }
}
```

To override just the subject for one particular rule while reusing the
template's body, add `"subject": "..."` alongside `template_id` in that
action's config — it wins over the template's subject.

The same applies to `notify.sms` templates (`channel: "sms"`, `body` holds
the message text, no `subject`).

---

## 3. notify.sms — send an SMS

Generic HTTP relay: point `notifications.sms.provider_url` (or the action's
own `config.provider_url`) at your gateway's webhook, or an adapter in front
of a vendor API (Twilio, Vonage, ...). CDS POSTs `{"to", "message"}` as JSON.

```bash
curl -sk -X POST "$CDS/t/carbon.super/cds/api/v1/orchestration-rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "otp-follow-up-sms",
    "trigger": { "event_type": "track", "event_name": "high_risk_login" },
    "actions": [
      {
        "type": "notify.sms",
        "config": {
          "to": "{{profile.identity_attributes.mobile}}",
          "message": "We noticed a new sign-in. Not you? Reply STOP to lock your account."
        }
      }
    ],
    "priority": 10,
    "is_active": true
  }'
```

---

## 4. profile.update — enrich the profile from the event

```bash
curl -sk -X POST "$CDS/t/carbon.super/cds/api/v1/orchestration-rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "track-last-cart-activity",
    "trigger": { "event_type": "track", "event_name": "add_to_cart" },
    "actions": [
      {
        "type": "profile.update",
        "config": {
          "patch": {
            "traits.last_cart_activity": "{{event.event_timestamp}}",
            "traits.last_added_product": "{{event.properties.object_name}}"
          }
        }
      }
    ],
    "priority": 20,
    "is_active": true
  }'
```

---

## 5. profile.search chained into notify.email

`profile.search` stores its result under `search_result` in the same rule's
execution context, so a later action can reference it.

```bash
curl -sk -X POST "$CDS/t/carbon.super/cds/api/v1/orchestration-rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "notify-account-owner-on-risk-event",
    "trigger": { "event_type": "track", "event_name": "high_risk_login" },
    "actions": [
      {
        "type": "profile.search",
        "config": { "by": "profile_id", "value": "{{event.profile_id}}" }
      },
      {
        "type": "notify.email",
        "config": {
          "to": "{{search_result.identity_attributes.emailaddress}}",
          "subject": "Security alert",
          "body": "A high-risk sign-in was detected on your account."
        }
      }
    ],
    "priority": 5,
    "is_active": true
  }'
```

---

## 6. profile.merge — force unification for an event's profile

Re-enqueues the profile onto the standard unification pipeline (the same
path a create/update triggers), evaluated against the org's configured
[unification rules](../concepts/unification-rules.md) — it does not bypass
or duplicate that logic.

```bash
curl -sk -X POST "$CDS/t/carbon.super/cds/api/v1/orchestration-rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_name": "reconcile-on-identify",
    "trigger": { "event_type": "identify", "event_name": "login_completed" },
    "actions": [
      { "type": "profile.merge", "config": {} }
    ],
    "priority": 10,
    "is_active": true
  }'
```

---

## Sending the triggering event

Any of the rules above starts running once a matching event is posted:

```bash
curl -sk -X POST "$CDS/t/carbon.super/cds/api/v1/events" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_id": "6f1b2c3d-...",
    "event_type": "track",
    "event_name": "add_to_cart",
    "application_id": "my-storefront",
    "properties": { "object_name": "Educational #2", "value": 49.65 }
  }'
```

The event is persisted synchronously (the response includes the assigned
`event_id`); rule matching and action execution happen asynchronously on the
orchestration queue.

Ingestion stays flat (`POST /events`) rather than nested under a profile,
since the caller often doesn't know the CDS-internal `profile_id` yet
(anonymous visitors are tracked by cookie and identified later). To read a
specific profile's event history, use the nested read endpoint instead:

```bash
curl -sk "$CDS/t/carbon.super/cds/api/v1/profiles/{profile_id}/events?limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Debugging: did my action fire?

```bash
curl -sk "$CDS/t/carbon.super/cds/api/v1/orchestration-rules/{rule_id}/executions?limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

Returns one row per action run, newest first, with `status`
(`success`/`failed`/`skipped`) and `error_message` when applicable — this is
the audit trail described in the concept doc.
