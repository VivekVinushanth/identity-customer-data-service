# Orchestration Engine

**The orchestration engine reacts to incoming events and, based on the event plus the current customer profile, executes a set of configured actions** — call the owning application (e.g. to trigger an in-app prompt or banner), send an email, send an SMS, update the profile, search for a profile, or merge profiles.

It is the "reactive" counterpart to the other rule engine CDS has:

| Engine | Answers |
|---|---|
| [Unification rules](unification-rules.md) | "Are these two profiles the same person?" |
| **Orchestration engine** | "Given this event and this profile, what should happen next?" |

Implemented: `POST/GET /events` (+ `GET /profiles/{profileId}/events`), full `/orchestration-rules` CRUD, full `/notification-templates` CRUD, asynchronous rule matching and action execution, and an execution audit trail. See [orchestration-examples.md](../guides/orchestration-examples.md) for worked curl examples of every action type.

---

## Where this fits with what already existed

Two pieces were already speculatively defined in `api/customer-data-service.yaml` with no `internal/` implementation before this engine was built:

- **`/events`** — now implemented in `internal/event/` (`model/store/service/provider/handler`), matching the schema that was already spec'd (`profile_id, event_type, event_name, application_id, org_handle, event_timestamp, properties, context`). `org_handle` is always resolved from the authenticated request path, never accepted from the client — a caller cannot spoof another tenant's events.
- **`ProfileEnrichmentRule`** (`/enrichment-rules`) — had the trigger/condition shape orchestration rules needed (`RuleTrigger{event_type, event_name, conditions: RuleCondition[]}`) but was hardwired to a single action. **This was left as-is, unimplemented** — `/orchestration-rules` was built as the generalized engine (same trigger/condition shape, pluggable `actions: Action[]`, with `profile.update` covering what enrichment rules would have done). Consolidating or deprecating `/enrichment-rules` in favor of it is still an open call — see below.

`profile.merge` and `profile.search` actions are **not** reimplementations — they call into `ProfilesServiceInterface` (`GetProfile`, `FindProfileByUserId`) and re-enqueue onto the existing unification pipeline (`internal/system/workers.EnqueueProfileForProcessing`), the same way a normal profile create/update does today.

---

## Core model

```
Event  ──▶  matches Trigger  ──▶  Conditions pass (event + profile)  ──▶  Actions run in order
```

### Trigger

```go
type Trigger struct {
    EventType string `json:"event_type"` // e.g. "track"
    EventName string `json:"event_name"` // e.g. "add_to_cart"
}
```

### Condition

```go
type Condition struct {
    Field    string `json:"field"`    // "event.properties.value" or "profile.traits.plan"
    Operator string `json:"operator"` // eq, neq, gt, gte, lt, lte, contains, exists
    Value    string `json:"value"`
}
```

Prefixing `field` with `event.`, `profile.` or `search_result.` selects which document is read. Conditions are flat AND-only — no OR/grouping (see Open Questions).

### Rule

```go
type OrchestrationRule struct {
    RuleId     string
    OrgHandle  string
    RuleName   string
    Trigger    Trigger
    Conditions []Condition
    Actions    []Action   // executed in array order
    Priority   int
    IsActive   bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

Multiple rules can match the same event (unlike unification rules, which stop at the first match) — an `add_to_cart` event might both fire a banner prompt *and* update a `last_cart_activity` trait. Rules run in ascending `priority` order; within a rule, actions run in `actions[]` array order. Rules sharing a trigger must have distinct priorities (enforced on create/update, same as unification rules).

### Action

```go
type Action struct {
    Type   string                 `json:"type"`   // see catalog below
    Config map[string]interface{} `json:"config"` // shape depends on Type
}
```

`Config` string values (including nested, including inside arrays) may contain `{{event.*}}`, `{{profile.*}}` or `{{search_result.*}}` placeholders, resolved by `internal/orchestration/matcher.ResolveTemplatesInConfig` at execution time. An unresolvable placeholder is left as literal text in the output rather than silently blanked — misconfiguration stays visible.

---

## Action catalog

All executors live in `internal/orchestration/executor/` and self-register via `init()` against a name → `Executor` registry (`executor.Register` / `executor.Get`), the same plugin pattern `internal/system/queue/factory.go` uses for queue providers.

| Type | Does | Config | Notes |
|---|---|---|---|
| `app.callback` | POSTs a JSON payload to an application-owned endpoint — the "prompt something / show a banner" hook | `endpoint_url` (required), `payload` (optional, defaults to `{rule_id, event, profile}`), `secret` (optional), `headers` (optional) | No `Application`-record lookup — the endpoint is supplied directly in config (adding a persisted per-application webhook URL is future work, see Open Questions). When `secret` is set, the request carries `X-CDS-Signature: sha256=<hmac>` over the raw body. |
| `notify.email` | Sends a plain-text email via SMTP | `to` (required); `template_id` and/or `subject`+`body` (one of the two required — see Templates below) | Uses Go's stdlib `net/smtp`. Server settings come from `notifications.smtp` in `deployment.yaml`; empty `host` makes the action fail with a clear error rather than attempt a connection. |
| `notify.sms` | Sends an SMS via a generic HTTP relay | `to` (required); `template_id` and/or `message` (one of the two required); `provider_url` (optional override) | POSTs `{"to","message"}` as JSON to `notifications.sms.provider_url` (or the per-action override). There's no single standard SMS gateway API, so this is deliberately a thin, bring-your-own-gateway relay rather than a specific vendor integration. |
| `profile.update` | Patches the target profile | `profile_id` (optional, defaults to the event's), `patch` (required, dotted-path → value map) | Calls `ProfilesService.PatchProfile`. This is what an enrichment rule's single action would have done. |
| `profile.search` | Looks up a profile and stores it under `search_result` for later actions in the same rule | `by` (`profile_id` \| `user_id`, default `profile_id`), `value` (default: the event's `profile_id`) | Calls `ProfilesService.GetProfile` / `FindProfileByUserId`. Synchronous, in-process — result never persisted, only held in the execution context. |
| `profile.merge` | Re-enqueues a profile onto the unification pipeline | `profile_id` (optional, defaults to the event's) | Calls `workers.EnqueueProfileForProcessing`, the same entrypoint a normal profile create/update uses — evaluated against the org's configured unification rules, not a separate merge path. |

---

## Templates

`notify.email` / `notify.sms` content doesn't have to be inlined in every rule. `internal/notification_template/` (`model/store/service/provider/handler`, `/notification-templates` REST surface) manages reusable, per-org, per-channel content: `{template_id, channel: "email"|"sms", name, subject (email only), body}`.

An action references one by `config.template_id`; `config.subject`/`config.body` (or `config.message` for SMS) — if also set — override the template's values field-by-field rather than requiring an all-or-nothing choice. Precedence is resolved in `internal/orchestration/executor/template.go` (`resolveNotificationContent`), shared by both executors. Either `template_id` or the inline content is required; the resolved subject/body still go through the same `{{event.*}}`/`{{profile.*}}`/`{{search_result.*}}` template-placeholder resolution as everything else in `config`.

This intentionally reuses the word "template" for two different things — `{{...}}` placeholder syntax (resolved at execution time against the event/profile) and a `NotificationTemplate` resource (reusable content, referenced by ID) — because a `NotificationTemplate`'s own `subject`/`body` are themselves placeholder templates.

---

## Execution pipeline

Mirrors the shape of the existing profile-unification pipeline — event ingestion is decoupled from processing via a queue, same as `ProfileUnificationQueue`:

1. **Ingest** — `POST /events` validates and persists the event (`internal/event`), then hands it to `internal/orchestration/worker.EnqueueEventForOrchestration`, which enqueues onto an `OrchestrationQueue` (`internal/system/queue`, same `Enqueue/Start/Close` shape as `ProfileUnificationQueue`; in-memory by default, pluggable via the same provider-registry pattern used for the other queues — only the in-memory provider ships today, see Open Questions).
2. **Match** — the orchestration worker (`internal/orchestration/worker`) fetches active rules for the event's `org_handle` whose trigger matches `event_type`/`event_name`, sorted by `priority` (`orchestration/store.GetActiveRulesForTrigger`).
3. **Evaluate** — the event's profile (if any) is loaded once via `ProfilesService.GetProfile` and reused across all matched rules. For each rule, `orchestration/matcher.EvaluateConditions` checks `Conditions` against `{event, profile}` (AND semantics).
4. **Execute** — passing rules run `Actions[]` in order via `executor.Execute`. A `profile.search` result is stored under `search_result` in that rule's own execution context so later actions in the same rule can reference it; it does not leak across rules.
5. **Record** — every action run (pass or fail) is written to `action_executions` as an `ActionExecution` row (`rule_id`, `event_id`, `action_index`, `action_type`, `status`, `error_message`, `attempt_count`, `executed_at`), queryable via `GET /orchestration-rules/{rule_id}/executions` — the "why didn't my banner fire" debugging path.

Failure isolation: one action failing (e.g. SMS provider down) does not block sibling actions or other rules — each action's outcome is recorded independently, and the worker continues to the next.

---

## Reliability — what's implemented vs. deferred

Implemented:
- **Failure isolation** across actions and rules (above).
- **Multi-tenancy**: `org_handle` scoping throughout — a rule only ever matches events and reads profiles within its own org, and the API rejects cross-org access via the same tenant dispatcher every other module uses.
- **Audit trail**: every action run recorded, independent of success/failure.

Deferred (no retry/backoff loop today — a failed external action is recorded as `failed` and not automatically retried; `attempt_count` exists in the schema for when this lands):
- Automatic retries with backoff for `app.callback` / `notify.email` / `notify.sms`.
- Dedup keyed on `(event_id, rule_id, action_index)` beyond what the audit log naturally provides for manual inspection.
- An external (ActiveMQ or otherwise) `OrchestrationQueue` provider — only in-memory exists; a production multi-instance deployment currently processes orchestration events per-instance rather than through a shared durable queue.

---

## Module shape (as implemented)

```
internal/event/                    # POST/GET /events, GET /profiles/{profileId}/events
  model/ store/ service/ provider/ handler/

internal/notification_template/    # /notification-templates CRUD
  model/ store/ service/ provider/ handler/

internal/orchestration/
  model/                            # OrchestrationRule, Trigger, Condition, Action, ActionExecution
  store/                            # rule persistence + action_executions audit log
  matcher/                          # condition evaluation + {{...}} template resolution (dependency-free)
  executor/                         # pluggable per-action-type executors (profile.*, app.callback, notify.*)
  service/                          # rule CRUD + validation
  provider/                         # DI seam for service
  handler/                          # /orchestration-rules REST surface
  worker/                           # consumes OrchestrationQueue, matches rules, dispatches actions

internal/system/queue/              # + OrchestrationQueue interface, alongside Profile/SchemaSync queues
```

`orchestration/worker` is a separate package from `internal/system/workers` (the existing profile/schema-sync/cookie-cleanup consumers): `orchestration/executor`'s `profile.merge` action calls back into `internal/system/workers.EnqueueProfileForProcessing`, so keeping the orchestration consumer loop in its own package avoids an import cycle between the two.

---

## Open questions (not yet decided)

1. **Enrichment rules**: `/enrichment-rules` remains unimplemented and un-deprecated. Decide whether to implement it as a thin view over `/orchestration-rules` (single `profile.update` action), or drop it from the spec now that orchestration rules supersede it.
2. **Condition language**: still flat AND-only. Revisit if a real rule needs OR/grouping.
3. **`app.callback` endpoint source**: currently supplied per-action in `config.endpoint_url`, not resolved from the `Application` record. Worth revisiting once applications have a registered callback URL of their own.
4. **Retry/backoff**: not implemented for external actions — see Reliability above.
5. **External `OrchestrationQueue` provider**: only in-memory ships; an ActiveMQ (or other broker) provider would need adding for durable, multi-instance delivery, following `docs/guides/extending-queue-providers.md`.
6. **Execution audit retention**: `action_executions` has no cleanup job yet (cf. `cookie_cleanup_worker.go` for the existing pattern to follow).
