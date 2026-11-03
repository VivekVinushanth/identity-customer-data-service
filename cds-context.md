# CDS / CDM — Full Context

> **Version: `v0.6.1+3178ceb`** — code-reconciled **2026-11-02**. Previous baseline: `v0.4.1+a9fdadd` (2026-10-26).  
> **Source-of-truth commit:** `3178ceb5c389910f47bd051ad799c5d32aebc821` (`wso2/identity-customer-data-service`, `main`); **previous source-of-truth:** `a9fdadd78e5b575865ee9ed0d2ee99af40d6b3ae`.  
> **Canonical living source:** `/WSO2-CDS/cds-context.md`; replace this file *in place*, never maintain an additional active context copy.  
> **Version semantics:** `version.txt = v0.6.1`; `install/helm/Chart.yaml version = 0.0.38` and `appVersion = "0.0.1"` (independent, stale placeholder). The `+3178ceb` qualifier identifies the reviewed code, **not** a release tag.
>
> **Authority & labels.** Go code/SQL/actual route registrations are authoritative when PR descriptions, repository docs or API specs disagree. Markers **Implemented**, **Operational caveat**, **Known gap**, and **Intent / roadmap** distinguish deployed-in-code behavior from future plans. GitHub merge does not by itself prove production deployment.
>
> **Latest reviewed delta (`a9fdadd` → `3178ceb`).** 84 commits including merges and release automation. Product changes span typed deterministic/fuzzy identity resolution, consent-sensitive review decisions, B2B child organization routing and resource sharing, org-isolated reads, finer-grained permissions, configuration/Helm, migrations, test coverage and Go toolchain security. The change is **not** a commit-history report; grouped intents and details follow in their relevant context sections. Key PRs: [#307](https://github.com/wso2/identity-customer-data-service/pull/307), [#334](https://github.com/wso2/identity-customer-data-service/pull/334), [#335](https://github.com/wso2/identity-customer-data-service/pull/335), [#336](https://github.com/wso2/identity-customer-data-service/pull/336), [#338](https://github.com/wso2/identity-customer-data-service/pull/338), [#339](https://github.com/wso2/identity-customer-data-service/pull/339), [#340](https://github.com/wso2/identity-customer-data-service/pull/340), [#341](https://github.com/wso2/identity-customer-data-service/pull/341), [#350](https://github.com/wso2/identity-customer-data-service/pull/350), [#320](https://github.com/wso2/identity-customer-data-service/pull/320), [#355](https://github.com/wso2/identity-customer-data-service/pull/355), [#356](https://github.com/wso2/identity-customer-data-service/pull/356). PRs without assignees use their authors as owners; `VivekVinushanth` for #307/#334/#335/#336/#338/#339/#341/#350 and `SujanSanjula96` for #320/#340/#355/#356.

---

## Table of Contents

1. [Product Vision & Strategic Context](#1-product-vision--strategic-context)
2. [Core Use Cases](#2-core-use-cases)
3. [Repository & Runtime Architecture](#3-repository--runtime-architecture)
4. [Routing & URL Structure](#4-routing--url-structure)
5. [Authentication, Authorization & Scopes](#5-authentication-authorization--scopes)
6. [Admin Config & Org Enablement](#6-admin-config--org-enablement)
7. [Applications & System Apps](#7-applications--system-apps)
8. [Data Model & Terminology](#8-data-model--terminology)
9. [Profiles](#9-profiles)
10. [Profile Schema](#10-profile-schema)
11. [Consent Management](#11-consent-management)
12. [Unification Rules](#12-unification-rules)
13. [How Profile Unification Works](#13-how-profile-unification-works)
14. [IS Sync — Identity Server Event Integration](#14-is-sync--identity-server-event-integration)
15. [Schema Sync](#15-schema-sync)
16. [Message Queue & Extending Providers](#16-message-queue--extending-providers)
17. [Database Schema](#17-database-schema)
18. [Deployment & Configuration](#18-deployment--configuration)
19. [Real-World Scenarios](#19-real-world-scenarios)
20. [MVP & Roadmap](#20-mvp--roadmap)

---

## 1. Product Vision & Strategic Context

> *Intent — from requirements material, not code.*

**We are NOT trying to enter the CDP market.** The goal is to incorporate the useful/possible parts of CDP needs into the WSO2 IAM product.

### Why build this?

- CDPs are bulky, expensive, and overbuilt for SMBs
- Existing CRM systems lack sufficient CDP capabilities
- Identity is the **foundation** of personalization — as a CIAM vendor, WSO2 is uniquely positioned to turn authentication and user context into real-time personalized experiences
- Enables "CIAM 3.0": connecting personalization and identity (the "digital double" story)
- Democratizes enterprise-grade personalization for SMBs without requiring a full CDP stack

### Who are we solving this for?

- **Developers** — want to integrate personalization quickly with minimal overhead
- **Product teams** — want engagement/conversion improvements without dedicated data teams
- **End users / customers** — expect modern, relevant product experiences from first interaction
- **Enterprises with siloed systems** — many systems holding fragmented customer data, wanting a Customer 360 view
- **Startups** — building new digital experiences, want to offload both IAM and customer data storage

### Positioning

> "WSO2's Customer Data Service brings customer context closer to where identity is already managed — at the point of authentication and access."

Key benefits:
- Unified, identity-first customer data management
- Omnichannel 360-degree customer data (web, mobile, in-person)
- Security, privacy, and consent-aware by design
- Developer-friendly APIs and SDKs for personalization
- Guest user profile management and anonymous-to-identified conversion tracking
- Cost-effective entry point (leverages existing IAM infrastructure)

### What CDS is NOT

- Not a full CDP (no segmentation engine, no campaign management OOTB)
- Not replacing IS — it works alongside IS
- Not handling merging of user **accounts** with different UserIDs (that's an IS responsibility) — and this is enforced in code: two permanent profiles with different `user_id` values are never merged

---

## 2. Core Use Cases

| Use Case | Description | Implemented? |
|---|---|---|
| **A1 — Identity Stitching** | Link anonymous user activity to registered accounts post-signup/login | Yes — cookie + `AUTHENTICATION_SUCCESS` / `POST_ADD_USER` |
| **A2 — Unifying Siloed Profiles** | Merge user profiles across devices, channels, applications | Yes — unification rules + worker |
| **B1 — Conversion Tracking** | Track anonymous-to-registered conversion | Partial — the lineage exists (`merged_from`), no analytics layer |
| **C1 — Master Data Management** | Single source of truth for customer profiles | Yes — profiles + schema + merge strategies |
| **D — Personalization** | APIs for app developers to use customer data to customize app behavior | Yes — consent-filtered profile reads, `/profiles/Me` |

---

## 3. Repository & Runtime Architecture

### Current architecture — 2026-11-02 (Implemented)

Two new domain families sit beside `profile`, `profile_schema`, `consent`, and `unification_rules`:

- `internal/identity_resolution/{engine,handler,model,provider,service,store,worker}`: rule-by-rule deterministic and tolerant comparisons; normalized exact/phonetic/LSH blocking; bounded candidate retrieval; score/verdict routing; pending review, resolution and durable rejection evidence. `internal/system/services/identity_resolution_service.go` registers the two review routes. It is integrated with profile/unification worker paths rather than a second independent matching system.
- `internal/organization/{handler,model,service,store,suborg}`, `internal/sharing/{handler,model,service,store}` and `internal/system/idp`: tenant-path org resolution, org-tree sync and events, policy reach and target validation, effective inherited/shared schema and rules, child enablement. The shared mux registers organization handlers only when `organization_management.enabled` is true; the HTTP tenant dispatcher itself understands both primary and `/o/{org_id}` paths.
- `internal/system/database/rows` normalizes database row/value handling; `internal/system/database/scripts/queries_organization.go` and `queries_sharing.go` add prepared statements; a bounded recursive ancestor query has a 64-hop limit. The `org_handle` tenant key remains fundamental for isolation.

**Dataflow:** `HTTP tenant dispatcher → token/org authorization → profile handler / worker → effective rules and schema → blocking candidate lookup → evidence scorer → auto-merge OR review/rejection`; child-org requests also traverse organization access and effective shared-resource policy checks. No separate standalone service, third-party ML scoring API, or new message-broker type is established by this delta.


| Property | Value |
|---|---|
| Repo | `wso2/identity-customer-data-service` |
| Language | Go — `go 1.26.9` in `go.mod`; README states Go 1.26+ |
| License | Apache-2.0 |
| Version pinned here | Snapshot label `v0.4.1+a9fdadd`; `version.txt` remains the published release `v0.4.1` (2026-10-14). The release builder owns version bumps (§18). |
| Last actual release | `v0.4.1` (published 2026-10-14; preceded by `v0.4.0` on 2026-10-13) |
| Datastore | **Two options.** Inbuilt SQLite (`datasource.type: sqlite`, **the default when unset**) or PostgreSQL (`postgres`). `dbscripts/sqlite.sql` is embedded and applied by the server; `dbscripts/postgres.sql` is applied by the operator. |
| SQLite driver | `modernc.org/sqlite v1.53.0` (pure Go — no CGO, no external server) |
| HTTP | Go stdlib `net/http` + `ServeMux` with method+pattern routing (`"GET /path/{id}"`) |
| Build | `make all` → `target/cds-<version>.zip` |
| Test targets | `make integration-test` (PostgreSQL, needs Docker), `make integration-test-sqlite` (`CDS_TEST_DB=sqlite`, no Docker), `make unit-test`, `make mq-integration-test` |
| Container | `Dockerfile`; Helm chart at `install/helm` |

### Top-level layout

```
.github/                GitHub Actions workflows and generated release-note categorization (§18)
api/                    OpenAPI spec (not generated from code; endpoint list is authoritative in §4)
cmd/server/             main.go — entrypoint, blank-imports queue providers
config/repository/conf/ deployment.yaml
dbscripts/              postgres.sql (operator-applied), sqlite.sql (embedded via embed.go)
docs/                   concepts/ + guides/
install/helm/           Chart, values, templates (deployment, svc, hpa, pdb, ingress)
internal/               all application code
scripts/local-setup/    script.sh + templates/ — scripted CDS + Identity Server dev setup (§18)
test/                   integration, activemq_integration, setup + scenario guides
```

### Domain modules under `internal/`

Each follows the same layering: `handler/` (HTTP) → `provider/` (DI) → `service/` (business logic) → `store/` (SQL) with `model/` for types.

| Module | Purpose |
|---|---|
| `profile` | Profile CRUD, cookies, IS sync endpoint, per-profile consents |
| `profile_schema` | Schema attribute CRUD + IS schema sync endpoint |
| `unification_rules` | Rule CRUD, priority/name uniqueness |
| `consent` | Consent category CRUD |
| `admin_config` | Per-org `cds_enabled`, `initial_schema_sync_done`, `system_applications` |
| `application` | Maps `client_id` ↔ `app_id` per org |
| `identity_provider` | Models for IS responses (applications list, claims) |
| `health_check` | Liveness/readiness |

### `internal/system/` (cross-cutting)

`authn` (JWT parse / introspection), `authz` (scope check), `security` (middleware, `AuthnAndAuthz`, `AuthnWithAdminCredentials`), `cache`, `client` (Identity Server client), `config`, `constants`, `database` (+ `client`, `model`, `provider`, `scripts`), `errors`, `log`, `managers` (service registration), `pagination`, `queue` (+ `inmemory`, `activemq`), `services` (route registration), `utils`, `workers`.

### Datasource abstraction & query layer (`internal/system/database/`)

Reworked in v0.3.63 to support two dialects. **Stores are datasource-agnostic**; the client is the only place that picks a dialect.

| File / package | Role |
|---|---|
| `constants.go` | `TypePostgres` / `TypeSQLite`, `DefaultType = TypeSQLite`, driver names, SQLite defaults **and PostgreSQL pool defaults** (`100` open / `25` idle / `30m` lifetime / `5m` idle / `10s` connect timeout) |
| `args.go` | `NormalizeSQLiteArgs` — binds `[]byte` args as text so marshalled JSON stays queryable by `json_extract` instead of landing as a BLOB |
| `model/dbquery.go` | `DBQuery{ID, Query, SQLiteQuery}` + `GetQuery(dbType)`, `Format(...)`, `Append(...)`, `WithSQL(...)` |
| `model/tx.go` | `Tx` carries `dbType` and exposes `ExecContext` / `QueryContext`; context-free `Exec` / `Query` remain only as deprecated wrappers |
| `client/dbclient.go` | `ExecuteQueryContext`, `BeginTxContext`, `DBType`, `NewSharedDBClient`; context-free `ExecuteQuery` / `BeginTx` are deprecated wrappers. `Close()` is always a no-op because the process owns the pool. |
| `client/normalize.go` | Coerces SQLite scan results to the Go types `lib/pq` returns, keyed on declared column type: `JSONB` string→`[]byte`, `BOOLEAN` `0/1`→`bool`, `TIMESTAMP` text→`time.Time` |
| `provider/dbprovider.go` | **One process-wide pool for either datasource.** First initialization is serialized; PostgreSQL is opened through `pq.NewConnector`, pool limits are applied and `PingContext` must succeed before the handle is published. `CloseDB()` closes the process-owned pool once; later `GetDBClient()` calls fail with `ErrDatabaseClosed`. `SetTestDB(db, dbType)` remains for tests. |
| `provider/bootstrap.go` | `ValidateDataSource` (server refuses to start on an unsupported type or, for `postgres`, missing hostname/username/password/name) and `EnsureDatabase` (creates the SQLite file + dir, applies the embedded schema in a transaction, with statement-by-statement fallback) |
| `scripts/query.go` | `newQuery(id, base, sqliteOverride...)`. `base` is the PostgreSQL text and the fallback for every other dialect |
| `scripts/queries.go` | All statements as `DBQuery` values (previously `map[string]string` keyed by db type), including bounded unification candidate lookups `CDS-PRF-19` / `CDS-PRF-20` |
| `scripts/registry.go` | `AllQueries()` — hand-maintained map of all **69** statements, so tests can prepare/validate each one on both datasources. A statement omitted here goes unprepared and unchecked. |
| `scripts/dialect.go` | Runtime SQL fragments that differ per dialect: `LikeOperator`, `JSONEqCondition`, `JSONLikeCondition`, `KeysetCondition`, `EncodeStringArray` / `DecodeStringArray`, `ValidateJSONKey` |
| `scripts/profile_lookup.go` | Builds datasource-specific bounded rule-candidate queries. Only `traits` / `identity_attributes` can become dynamic SQL columns; nested attribute paths and candidate values remain bound data. PostgreSQL uses JSONPath plus an exact string-element check; SQLite uses recursive JSON traversal with equivalent semantics. |

**Statement ID convention:** `CDS-<DOMAIN>-<NN>` where DOMAIN ∈ `APP` (2), `SCH` (15), `UNR` (5), `PRF` (19 registered statements; IDs now extend through `CDS-PRF-20` after one older full-master lookup was removed), `CON` (15), `CKI` (8), `CFG` (4), `SYS` (1) — 69 total. IDs are permanent — new statements take the next unused number rather than renumbering. IDs appear in query error messages (`query CDS-PRF-07 failed: ...`).

**Migration note for readers of the old context:** the store idiom `scripts.X[provider.NewDBProvider().GetDBType()]` is gone. Stores pass `scripts.X` (a `DBQuery`) to the database client. As of `49c5abc`, request-driven stores use `ExecuteQueryContext(ctx, ...)` / `BeginTxContext(ctx)` so cancellation reaches both the wait for a pooled connection and the database operation itself. `dbClient.DBType()` is called only where bind *arguments* differ, not to select a statement.

**Dialect differences the layer papers over:**

| Concern | PostgreSQL | Inbuilt SQLite |
|---|---|---|
| Case-insensitive match | `ILIKE` | `LIKE` (ASCII-only case folding) |
| JSON equality | `col @> $n::jsonb` (containment, keeps GIN usable) | `json_extract(col, '$."key"') = $n` |
| JSON pattern match | `col -> 'a' ->> 'b' ILIKE $n` | `json_extract(col, '$."a"."b"') LIKE $n` |
| Keyset pagination | `(p.created_at, p.profile_id) < ($n::timestamptz, $m::text)` | same row-value comparison, no casts |
| String list column (`consent_categories.destinations`) | `TEXT[]` | JSON array text |

`ValidateJSONKey` restricts interpolated JSON keys to `constants.FilterRegex`, since filter keys are user-supplied and are not bindable as parameters. Before v0.3.63 these keys were `fmt.Sprintf`'d into the statement raw, so this is a SQL-injection hardening fix as well as a dialect helper. **It is also a user-visible behaviour change:** a `co` / `sw` / `eq` filter on a key that fails the regex now returns a `FILTER_PROFILE` error ("Invalid filter key: …") instead of reaching the database. Applies to profile filters (§9) and profile-schema attribute filters (§10).

`HealthCheckPing` (`CDS-SYS-01`) replaces the inline `"SELECT 1;"` previously in `health_check/service/health_check_service.go`. The **readiness** handler now passes `r.Context()` to `CheckReadiness`, and the database query uses `ExecuteQueryContext`; if Kubernetes abandons/times out the probe, the wait for a pooled connection and the query are cancelled too. Liveness remains independent of database reachability.

### Bounded unification candidate lookup (new since `71fb3bd`)

Profile unification no longer retrieves the full set of reference profiles for an organization. Candidate discovery is now expressed as two bounded store operations:

- `FindOldestReferenceProfileIDByUserID(ctx, org, userID, excludedProfileID)` → `CDS-PRF-19`
- `FindOldestReferenceProfileIDByAttributeValues(ctx, org, property, values, excludedProfileID, incomingUserID)` → dynamically completed `CDS-PRF-20`

Both queries return **at most one profile ID**, restricted to `REFERENCE_PROFILE` rows in the same organization and excluding the incoming profile. Selection is deterministic: `ORDER BY p.created_at, p.profile_id LIMIT 1`, so the oldest eligible master wins and profile ID breaks timestamp ties. Rule candidates also apply user-ID compatibility in SQL: when the incoming profile has a non-empty user ID, candidates with a different non-empty user ID are excluded before `LIMIT 1`, preventing an incompatible older profile from hiding a younger valid candidate.

Rule matching is deliberately string-based and exact. `unificationRuleValues` extracts distinct strings from `traits.*` or `identity_attributes.*`; a scalar string or an immediate string element of an array can match, including the empty string. Matching is case-sensitive; numeric, boolean, null, object leaves, and nested arrays do not match. Arrays of objects are traversed without consuming a path component, so nested object fields can still be matched. `application_data.*` is not supported as a candidate-lookup root by this implementation.

On PostgreSQL, the generated rule lookup uses an indexable JSONPath `@?` prefilter plus an exact `jsonb_path_query` / string-element check so existing GIN indexes on `traits` and `identity_attributes` remain eligible. On SQLite, an equivalent recursive `json_each` traversal preserves the same matching semantics but has no GIN acceleration. Only the two JSON column names are interpolated and they are selected from a fixed allowlist; paths and values are passed as query arguments, including quotes and JSON/SQL metacharacters.

### Request / operation context propagation (new since `b38e9bd`)

HTTP handlers now capture `ctx := r.Context()` and carry it through service → store → database calls across profiles, schemas, consents, unification rules, applications and admin config. The database client adds `ExecuteQueryContext` and `BeginTxContext`, and the `Tx` wrapper adds `ExecContext` / `QueryContext`. The old context-free methods remain as deprecated compatibility wrappers around `context.Background()`.

There is deliberately **no global database query timeout** at this layer: the caller owns the deadline. For HTTP requests that is the request context; for readiness it is the probe request; queue jobs have worker-owned contexts. `database/sql` automatically rolls a transaction back if the context supplied to `BeginTxContext` ends first.

### Process-owned database pools (new since `b38e9bd`)

Both SQLite and PostgreSQL are now process-scoped shared pools. A store's `DBClient.Close()` does not close anything; `provider.CloseDB()` owns final shutdown. For PostgreSQL the defaults per process/pod are `max_open_conns=100`, `max_idle_conns=25`, `conn_max_lifetime_seconds=1800`, `conn_max_idle_time_seconds=300`, `connect_timeout_seconds=10`. Zero/omitted values select defaults; negative values, or an idle limit above the resolved open limit, fail startup. Operators therefore need to budget `max_open_conns × maximum simultaneous pods` (including rollout surge) against PostgreSQL capacity.

### Background workers (`internal/system/workers/`)

| Worker | File | Trigger |
|---|---|---|
| Profile unification | `profile_worker.go` | Consumes `ProfileUnificationQueue` |
| Schema sync | `schema_sync_worker.go` | Consumes `SchemaSyncQueue` |
| Cookie cleanup | `cookie_cleanup_worker.go` | Timer — `cleanup.cookie.interval` (default 86400s), `batch_size` 500 |

`internal/system/workers/lifecycle.go` now gives queue workers an explicit job lifecycle. Once stopping begins, new jobs are refused; active jobs are waited on; if the shared shutdown deadline expires, their contexts are cancelled and shutdown is reported as incomplete. The cookie-cleanup worker also exposes a waitable stop rather than returning while its sweep is still running.

**Shutdown order:** stop HTTP intake → stop/drain the three workers concurrently → close the shared database pool last. The entire sequence shares one context derived from `shutdown.grace_period_seconds` (default **25s**). The Helm pod outer deadline is **30s**, intentionally longer than the application deadline. If the application deadline expires, unfinished job contexts are cancelled and the pool closes anyway; logs explicitly report incomplete work.

---

## 4. Routing & URL Structure

### Current path additions (Implemented; supersedes primary-only statements below)

- **Primary:** `/t/{primary_handle}/cds/api/v1/...`; **child organization:** `/t/{primary_handle}/o/{org_id}/cds/api/v1/...`. The dispatcher resolves the child `org_id` into its actual `org_handle`, stores both the effective handle and org ID in context, and strips the prefix before passing to the shared route mux. A child handle on a primary path is rejected; an unknown/mismatched child identifier is not permitted. `ExtractOrgBasePath` retains the correct prefix for constructed links.
- **Review queue:** `GET /identity-resolution/review-tasks`; **resolve:** `POST /identity-resolution/review-tasks/{taskId}/resolve`. These are implemented in `internal/system/services/identity_resolution_service.go` and handlers. Resolve carries approval/rejection; a successful approval may irreversibly unify profiles.
- **Primary-only organization APIs:** `POST /organizations/event-sync` (configured admin Basic); `POST /organizations/sync`; `GET /organizations/sync-status`; CRUD on `/organizations/access-policies[/{policyId}]` (normal OAuth organization operations). The event handler and tree-sync workflow update the local org graph.
- **Sharing policy APIs:** CRUD on `/profile-schema/{scope}/{attributeId}/sharing-policies[/{policyId}]` and `/unification-rules/{ruleId}/sharing-policies[/{policyId}]`; child organizations are not allowed to modify primary-owned sharing/config/schema/rule policy surfaces.
- **Primary-only health:** non-tenanted `/cds/api/v1/health` and `/cds/api/v1/ready` are unchanged.

**Caution:** The older route tables below are still useful for the original APIs but are **not exhaustive** for organization-management and identity-resolution routes. API spec entries are supplementary; the registered Go handlers are authoritative.


Routes are registered in `internal/system/services/*.go` against a shared mux, then mounted by `internal/system/managers/servicemanager.go`.

- `constants.ApiBasePath = "/cds/api"`; every service uses `base = ApiBasePath + "/v1"` → **`/cds/api/v1`**
- Tenanted routes are mounted behind a tenant dispatcher: **`/t/{orgHandle}` + `/cds/api/v1/...`** for primaries (child route includes `/o/{org_id}`)
  `MountTenantDispatcher` strips `/t/{orgHandle}`, injects the handle into request context under `TenantContextKey`, and delegates. Handlers read it via `utils.ExtractOrgHandleFromPath(r)`, falling back to `carbon.super`.
- Health/readiness are **non-tenanted**, served from a separate root mux under `/cds/`.

**Primary full path shape:** `https://{host}/t/{orgHandle}/cds/api/v1/{resource}`; child path includes `/o/{org_id}` as described above.

Confirmed by the `Location` header built in `InitProfile`:
`{serverURL}/t/{orgHandle}/cds/api/v1/profiles/{profileId}`

### Registered endpoints (code-derived)

**Health** — `internal/system/services/health_check_service.go` (non-tenanted)

| Method | Path |
|---|---|
| GET | `/cds/api/v1/health` |
| GET | `/cds/api/v1/ready` |

**Profiles** — `profile_service.go`

| Method | Path | Handler |
|---|---|---|
| GET | `/profiles` | `GetAllProfiles` — cursor pagination, filters, attribute projection |
| POST | `/profiles` | `InitProfile` — creates anonymous profile + sets cookie |
| GET | `/profiles/Me` | `GetCurrentUserProfile` — resolved via `cds_profile` cookie |
| PATCH | `/profiles/Me` | `PatchCurrentUserProfile` |
| POST | `/profiles/sync` | `SyncProfile` — IS event webhook (Basic admin auth) |
| POST | `/profiles/link` | `LinkProfile` — attaches a `user_id` to an anonymous profile (scope `profile:link`) **← new** |
| GET | `/profiles/{profileId}` | `GetProfile` |
| PATCH | `/profiles/{profileId}` | `PatchProfile` |
| DELETE | `/profiles/{profileId}` | `DeleteProfile` |
| GET | `/profiles/{profileId}/consents` | `GetProfileConsents` |
| PUT | `/profiles/{profileId}/consents` | `UpdateProfileConsents` |

> There is a `UpdateProfile` method on the handler that is **not routed** — `PATCH` is the only update path.
> There is **no** `/profiles/merge` and **no** `/profiles/{id}/merge-history` endpoint. Merge lineage is exposed inline on the profile as `merged_from` / `merged_to`.

**Profile Schema** — `profile_schema_service.go`

| Method | Path | Notes |
|---|---|---|
| GET | `/profile-schema` | Whole schema (all scopes) |
| DELETE | `/profile-schema` | Deletes the org's whole schema → `204` |
| POST | `/profile-schema/sync` | IS schema event webhook (Basic admin auth) |
| POST | `/profile-schema/{scope}` | Add attributes to a scope → `201` |
| GET | `/profile-schema/{scope}` | List attributes in a scope |
| DELETE | `/profile-schema/{scope}` | Delete all attributes in a scope → `204` |
| GET | `/profile-schema/{scope}/{attrID}` | Single attribute |
| **PUT** | `/profile-schema/{scope}/{attrID}` | Update single attribute → `200` (handler is named `Patch…` but the verb is `PUT`) |
| DELETE | `/profile-schema/{scope}/{attrID}` | Delete single attribute → `204` |

**Unification Rules** — `unification_rules_service.go`

| Method | Path |
|---|---|
| POST | `/unification-rules` |
| GET | `/unification-rules` |
| GET | `/unification-rules/{ruleId}` |
| PATCH | `/unification-rules/{ruleId}` |
| DELETE | `/unification-rules/{ruleId}` |

**Consent Categories** — `consent_service.go`

| Method | Path |
|---|---|
| GET | `/consent-categories` |
| POST | `/consent-categories` |
| GET | `/consent-categories/{categoryId}` |
| PUT | `/consent-categories/{categoryId}` |
| DELETE | `/consent-categories/{categoryId}` |

**Admin Config** — `admin_config_service.go`

| Method | Path |
|---|---|
| GET | `/config` |
| PATCH | `/config` |

---

## 5. Authentication, Authorization & Scopes

### Current auth and permissions (Implemented; source `security/middleware.go` and handler calls)

- Normal `AuthnAndAuthz` validates the bearer token against the **effective org handle** and, on a child path, the path `org_id` claim. Authorization uses `org:`-prefixed operation names for child routes; `AuthnAndAuthzPrimary` forbids child-path access to primary-only management operations. A missing `required_scopes` mapping denies access (403), not an implicit allow.
- Review **read**: `identity_resolution:view` → `internal_cds_identity_resolution_view`; review **decision**: `identity_resolution:update` → `internal_cds_identity_resolution_update`. On child routes the corresponding `org:` variants are configured (`org:identity_resolution:view` / `org:identity_resolution:update`). The old `identity_resolution:review` scope is superseded; custom installations using it must register the new scopes.
- Sharing-policy **read** no longer uses the generic schema/rule view scope: `profile_schema:share_view` → `internal_cds_profile_schema_share_view`; `unification_rules:share_view` → `internal_cds_unification_rule_share_view`. Policy **create/update/delete** uses `profile_schema:share_update` → `internal_cds_profile_schema_share_update` and `unification_rules:share_update` → `internal_cds_unification_rule_share_update`. Old `*:share` mappings are superseded. Suborg share-policy GET paths currently lack equivalent `org:*:share_view` settings and therefore return 403 even when ordinary resource view is authorized.
- `internal_org_cds_profile_link` is still defined on `/o/cds/api/v1/profiles` API resource for app authorization, but as of [#356](https://github.com/wso2/identity-customer-data-service/pull/356) it is **not** listed under local-setup `org_customerData` API resource collection. No corresponding primary `customerData` collection link scope exists.
- **Known gap:** the Helm chart's `auth_server.required_scopes` is missing the `consent_category:*` mappings present in the product config. Unchanged chart deployments can 403 on those APIs until chart authorization config is completed; investigate before rollout rather than assuming production parity.


Two distinct auth paths in `internal/system/security/middleware.go`:

| Path | Used by | Mechanism |
|---|---|---|
| `AuthnAndAuthz(r, "<scope-key>")` | All normal API endpoints | OAuth2 Bearer token |
| `AuthnWithAdminCredentials(r)` | `/profiles/sync`, `/profile-schema/sync` | HTTP Basic with configured admin credentials |

### Bearer token handling

`internal/system/authn/auth.go` supports **both** JWT and opaque tokens:
- JWT → parsed locally (`IsJWT`, `ParseJWTClaims`); claims of interest: `azp`, `client_id`, `active`, `org_handle`, `aud`, `exp`
- Opaque → introspected against IS via `introspection_client_id` / `introspection_client_secret`

`internal/system/authz/authorization.go` resolves the required scope strings from `auth_server.required_scopes` in `deployment.yaml`, so the scope mapping is **configuration, not hardcoded**.

### Scope map (from `config/repository/conf/deployment.yaml`)

| Scope key | Token scope |
|---|---|
| `profile:create` | `internal_cds_profile_create` |
| `profile:update` | `internal_cds_profile_update` |
| `profile:view` | `internal_cds_profile_view` |
| `profile:delete` | `internal_cds_profile_delete` |
| `profile:link` | `internal_cds_profile_link` **← new** |
| `unification_rules:view` | `internal_cds_unification_rule_view` |
| `unification_rules:create` | `internal_cds_unification_rule_create` |
| `unification_rules:update` | `internal_cds_unification_rule_update` |
| `unification_rules:delete` | `internal_cds_unification_rule_delete` |
| `profile_schema:view` | `internal_cds_profile_schema_view` |
| `profile_schema:create` | `internal_cds_profile_schema_create` |
| `profile_schema:update` | `internal_cds_profile_schema_update` |
| `profile_schema:delete` | `internal_cds_profile_schema_delete` |
| `admin_config:view` | `internal_cds_admin_config_view` |
| `admin_config:update` | `internal_cds_admin_config_update` |
| `consent_category:view` | `internal_cds_consent_category_view` |
| `consent_category:create` | `internal_cds_consent_category_create` |
| `consent_category:update` | `internal_cds_consent_category_update` |
| `consent_category:delete` | `internal_cds_consent_category_delete` |

### Setup prerequisites (from README)

1. Register an M2M application in the IS console, subscribed to Claim Management APIs, JWT access tokens, audience `iam-cds`
2. Create an admin user and add as console administrator
3. `conf/repository/config/dev.env` with `ENV`, `LOG_LEVEL`, `AUTH_SERVER_CLIENT_SECRET`, `AUTH_SERVER_ADMIN_PASSWORD`
4. Register the consuming applications and subscribe them to Profiles / Profile Schema / Unification Rules / Consent Management API resources
5. Obtain a client-credentials token with only the needed scopes

> **New in v0.3.65 — `scripts/local-setup/script.sh` automates all of the above**, and more besides. The README's five steps are *necessary but not sufficient* for a working IS↔CDS integration: the script also deploys the four extension bundles into IS `dropins` with their `[[event_handler]]` module config, exchanges TLS certificates in both directions, adds the `http://wso2.org/claims/cdsProfile` local claim and its SCIM2 mapping, sets `[console.extensions] cds_host`, relaxes the `/oauth2/introspect` policy, and flips `cds_enabled`. See §18. A setup built by following the README alone will authenticate but will not sync users.
>
> Two OAuth applications, not one: a **system app** for the calls CDS makes into IS, and a **client app** with `aud=iam-cds` for calling the CDS APIs. The Console is wired separately because it issues opaque tokens.
>
> The `internal_cds_*` scopes **cannot be created through the IS management API** — IS reserves the `internal_` prefix and rejects it over REST. The script seeds the CDS API resources and their scopes through `deployment.toml` instead.

---

## 6. Admin Config & Org Enablement

### Current matching and child enablement settings (Implemented)

| Key | Current default / behavior |
|---|---|
| `auto_merge_enabled` | `true` if unset; explicit `false` routes qualifying matches to review rather than auto-merge |
| `auto_merge_threshold` | **`1.0`** if unset (raised from `0.95`); only perfect final scores auto-merge by default |
| `manual_review_threshold` | `0.75` if unset; scores at/above enter review when not auto-merging |
| `deterministic_match_decisive` | `true` if unset, preserving historic exact-match auto-unification |
| `identity_resolution.fuzzy_matching_enabled` | server-wide `false` by default (deployment / Helm) |
| `identity_resolution.allow_evidence_strength_override` | server-wide `false` by default; client-provided strengths rejected until enabled |
| `organization_management.enabled` | `false` by default; when true requires the organization schema and IDP integration |

`PATCH /config` checks `[0,1]` thresholds (zero is an **unset** sentinel) and `0.3 < manual_review_threshold <= auto_merge_threshold - 0.01` for effective values. Values >1 are rejected rather than accepted but ignored. Suborganizations load **primary-organization matching thresholds and auto-merge policy**. A child org is operationally enabled only when the primary CDS feature is enabled, its org record is active and organization-access policy reaches it; tree sync must be completed before policy setup.


Model — `internal/admin_config/model`:

```go
type AdminConfig struct {
    OrgHandle             string   `json:"org_handle"`
    CDSEnabled            bool     `json:"cds_enabled"`
    InitialSchemaSyncDone bool     `json:"initial_schema_sync_done"`
    SystemApplications    []string `json:"system_applications"`
}
```

API surface is narrower than the internal model:
- `GET /config` → `AdminConfigAPI` = `{ cds_enabled, system_applications? }`
- `PATCH /config` → `AdminConfigUpdateAPI` = `{ cds_enabled *bool, system_applications? }` (pointer so absence ≠ false)

`initial_schema_sync_done` is system-managed and not exposed for write.

Persisted in the `cds_config` table as `(org_handle, config, value)` rows keyed by the constants `cds_enabled`, `initial_schema_sync_done`, `system_applications`.

### The `cds_enabled` gate

**Every** tenanted handler calls `isCDSEnabled(orgHandle)` before doing work. If false the request fails with `400` and error code `CDS_NOT_ENABLED`. This includes the IS sync webhook — sync events for a disabled org are rejected, not silently dropped.

---

## 7. Applications & System Apps

The `applications` table maps `(org_handle, client_id) → app_id`.

### Application identifier mode

`Config.ApplicationIdentifierType` (`application_identifier_type` in YAML):

| Value | Behaviour |
|---|---|
| `client_id` | **Default.** The OAuth `azp`/`client_id` claim *is* the application identifier |
| `app_id` | The claim is resolved through `ResolveAppIdentifierByClientID(orgHandle, clientID)` to an internal app ID |

`Config.UsesAppIDIdentifier()` is the helper that switches this.

### System app detection

1. `getCallerClientIDFromRequest(r)` — pull `azp`, falling back to `client_id`, from the JWT claims or introspection response
2. Resolve to an app identifier per the mode above
3. `isCallerSystemApplication(orgHandle, appId)` → `AdminConfigService.IsSystemApplication`, which checks membership in `system_applications`

> The `SystemApp` constant is an **outbound** `Authorization` scheme used by the Identity Server client (`internal/system/client/identity_client.go`), *not* an inbound request header. System-app status is never asserted by the caller — it is derived from the token.

### Application data filtering

`internal/profile/service/application_data_filter_service.go`, driven by query params:

| Param | Effect |
|---|---|
| `includeApplicationData=true` | **Required** — without it `application_data` is returned empty, for every caller |
| `application_identifier=a,b` or `*` | System apps only — selects which apps' data to include; `*` or omitted returns all |

| Caller | Result |
|---|---|
| System app | All app data, or the subset named in `application_identifier` |
| Regular app | Only its own `application_data[callerAppID]` bucket |

---

## 8. Data Model & Terminology

### Current model extensions — code reconciled

Identity resolution adds `blocking_keys` (normalized exact, Double Metaphone and LSH band keys), `review_tasks` (`match_score`, `status`, `match_reason`, `score_breakdown`, resolver metadata) and `rejection_pairs` (canonical ordered pair, score/breakdown at rejection, actor/time). Rules gain `attribute_type`, `unification_method`, nullable `match_strength`/`mismatch_strength`; `NULL` strengths mean derive from type. Derived `origin`, `owner_org_handle`, `state` on effective shared read projections are not persisted to rule rows.

B2B adds `organizations` (`org_id`, `org_handle`, `parent_org_id`, `primary_org_id`, status), `resource_sharing_policy`, `resource_sharing_policy_target`. **SQL names are `resource_sharing_policy*`, NOT `cds_share_policy*`**, contrary to one PR's draft description. Ownership and initiator are explicit IDs; targets encode ALL_CHILDREN / ORG / ORG_SUBTREE reach.

`org_handle` filtering now participates in more profile, consent-category, rule, edit and delete queries (especially PR #340); accidental resource-ID knowledge must not cross org boundaries. For child effective schema/rules, local and shared resources are resolved through owner/ancestor and policy-state rules, not copied into each child as durable duplicates.


### Current model

```
UserID ←  0..1 : 1  → ProfileID
Application ← 1 : 1 → ApplicationContext
ProfileID ← 1 : n → ApplicationContext
ApplicationContext ← 1 : n → (name, value)
```

Internally, profile relationships are stored in the `profile_reference` table with a `profile_status` state machine, and projected onto `merged_from` / `merged_to` at the API boundary.

### Key terminology

| Term | Definition |
|---|---|
| **Temporary Profile** (anonymous) | A profile with no `user_id`. Created on first anonymous interaction. Linked to a browser cookie. |
| **Permanent Profile** (identified) | A profile with a `user_id`. Acts as the master record. |
| **Reference Profile** | Code-level term for a master profile — `profile_status = REFERENCE_PROFILE`, `is_reference_profile = true`. Child profiles carry `MERGED_TO`. |
| **ApplicationContext / application_data** | Data for a given application and profile, stored as key-value pairs. |
| **Identity Attributes** | Attributes sourced from IS claims (email, phone, name…). |
| **Traits** | Behavioural/preference data managed by the application. |
| **Profile Unification** | Recognising two profiles as the same person and merging them (CDP "identity resolution"). |
| **Merge Reason** | Either `system:user_id_match` or the `rule_name` of the rule that fired. |
| **Anonymous Profile Tracker** | The `cds_profile` cookie value, echoed back on profile-init/Me responses as `anonymous_profile_tracker`. |

### Profile status constants (`internal/system/constants`)

`REFERENCE_PROFILE`, `WAIT_ON_ADMIN`, `WAIT_ON_USER`, `MERGED_TO`

The `WAIT_ON_*` states and the `ProfileStatus.IsWaitingOnAdmin` / `IsWaitingOnUser` flags exist in the model for the planned approval-workflow feature but are **not yet driven by the unification worker**.

### Merge use-case / mode constants

Merge use cases: `TEMP_TEMP`, `TEMP_PERM`, `PERM_PERM`
Merge modes: `MERGE_BY_ADMIN`, `MERGE_BY_USER`, `MERGE_ON_TRIGGER`
Sync triggers: `SYNC_ON_SCHEDULE`, `SYNC_ON_UPDATE`

These back the `profile_unification_modes` / `profile_unification_triggers` tables and `unification_rules/model.DefaultConfig()` (all three merge types default to `MERGE_ON_TRIGGER`, trigger type `SYNC_ON_UPDATE`). Not yet surfaced through any API.

---

## 9. Profiles

### Current isolation and identity-resolution integration (Implemented)

- Single-record reads/updates/deletes and downstream retrievals pass `(profileId, orgHandle)` into stores; `GetProfile` / `GetProfileWithOptions` require an explicit org handle. PR #340 also scopes consent-category reads, selected rule reads/deletes and rejected/approved review resolution lookups, with SQLite/PostgreSQL integration cases for cross-org access. This is important when IDs are globally unique but knowledge of an ID must not confer access.
- Creation or updates affecting matchable values can schedule identity resolution; re-evaluation occurs only when matched values actually change. The worker fetches fresh org-scoped profile state, performs bounded candidate selection, evaluates active *effective* rules, filters unmergeable permanent profiles with different user IDs, then merges or inserts review tasks. It retains existing merge-shape rules (permanent vs temporary and master/reference lineage), and attributes merge reasons to the decisive matching rule when determinable.
- Reviewing, cancelling, rejecting or re-opening tasks is stateful; rejection evidence is durable across routine edits and reconsidered candidates go to human review, not automatic union.


A **profile** is the central entity — a person's identity attributes, behavioural traits, and per-application data, unified across interactions.

### Internal model (`internal/profile/model/profile.go`)

```go
type Profile struct {
    ProfileId          string
    UserId             string
    OrgHandle          string
    CreatedAt          time.Time
    UpdatedAt          time.Time
    Location           string
    IdentityAttributes map[string]interface{}
    Traits             map[string]interface{}
    ApplicationData    []ApplicationData
    ProfileStatus      *ProfileStatus
}

type ProfileStatus struct {
    ReferenceProfileId string
    IsReferenceProfile bool
    IsWaitingOnAdmin   bool
    IsWaitingOnUser    bool
    DeleteProfile      bool
    ListProfile        bool
    ReferenceReason    string
    References         []Reference
}

type Reference struct {
    ProfileId string
    Reason    string
}
```

### API projection (`internal/profile/model/profile_api.go`)

`ProfileResponse`:

| Field | Notes |
|---|---|
| `profile_id` | |
| `user_id` | omitted when empty (anonymous) |
| `anonymous_profile_tracker` | cookie ID; set only by `InitProfile` and `GetCurrentUserProfile` |
| `meta` | `{ created_at, updated_at, location }` |
| `identity_attributes` | |
| `traits` | |
| `application_data` | `map[appId]map[key]value` — **flattened**, unlike the internal array form |
| `merged_to` | `Reference` — set on child profiles |
| `merged_from` | `[]Reference` — set on master profiles |

`ProfileListResponse` is the same minus `anonymous_profile_tracker` and `merged_to`.

`ProfileRequest` (write): `{ user_id, identity_attributes?, traits?, application_data }`. `InitProfile` decodes with `DisallowUnknownFields()` — unknown keys are a `400`.

`ProfileListAPIResponse`: `{ pagination, profiles }`.

`ProfileLinkRequest` / `ProfileLinkResponse` (new) — both are the same flat pair, and both fields are required on the request:

```go
type ProfileLinkRequest struct {
    ProfileId string `json:"profile_id" bson:"profile_id"`
    UserId    string `json:"user_id"    bson:"user_id"`
}

type ProfileLinkResponse struct {
    ProfileId string `json:"profile_id" bson:"profile_id"`
    UserId    string `json:"user_id"    bson:"user_id"`
}
```

> Note the `bson` tags on both. They are inert — CDS has no Mongo datasource (§18 lists only SQLite and PostgreSQL) — but they match the convention used by the surrounding DTOs in the same file.

### Temporary vs permanent

**Temporary (anonymous)** — no `user_id`; linked to the `cds_profile` cookie via `profile_cookies`; accumulates behavioural data.
**Permanent (identified)** — has `user_id`; acts as master; may absorb temporary profiles over time.

### Lifecycle

```
Anonymous visit
      │
      ▼
POST /profiles → temporary profile created, cds_profile cookie issued
      │
      │  (login — AUTHENTICATION_SUCCESS, or signup — POST_ADD_USER)
      ▼
userId attached to the temp profile
      │
      ▼
Unification worker picks it up
      │
      ├── No existing permanent profile → temp profile promoted to permanent
      │
      └── Existing permanent profile found → temp merged in as child
                │
                ▼
            master updated with merged data; child gets merged_to
                │
                │  (SESSION_TERMINATE)
                ▼
            cookie deactivated (is_active = false)
            cookie cleanup worker batch-deletes after 24h
```

> As of `b38e9bd` the `userId attached to the temp profile` step has a **second entry point**: a caller can drive it directly with `POST /profiles/link` instead of waiting for the IS event. Everything downstream of that step — the unification worker, promotion, merge, cookie teardown — is identical either way. See below.

### Explicit linking — `POST /profiles/link` (new)

Until now the only way a `user_id` reached an anonymous profile was implicitly, through the IS event webhook (`POST /profiles/sync` on `AUTHENTICATION_SUCCESS` / `POST_ADD_USER`, §14). `POST /profiles/link` adds a **direct, caller-driven** path for the same transition — useful when the caller already knows both IDs and does not want to wait for, or depend on, an IS event.

Handler: `ProfileHandler.LinkProfile` in `internal/profile/handler/profile_handler.go`.

Request `{ "profile_id": "...", "user_id": "..." }` → response `{ "profile_id": "...", "user_id": "..." }` on `200`.

**Order of checks in the handler** — this ordering is worth knowing, because it decides which error a bad call gets back:

1. `security.AuthnAndAuthz(r, "profile:link")` — Bearer token, scope `profile:link`. **Authz runs before the org gate**, unlike nothing else in particular; it is simply first.
2. `isCDSEnabled(orgHandle)` — `400 CDS_NOT_ENABLED` if the org has not enabled CDS (§6).
3. JSON decode — `400` with a decode description. Note this handler uses a **plain `json.NewDecoder(r.Body).Decode`, *not* `DisallowUnknownFields()`** as `InitProfile` does, so unknown keys in a link request are silently ignored rather than rejected. That is an inconsistency with the rest of the profile write surface, not a documented choice.
4. `profile_id` empty after `TrimSpace` → `400`, description `profile_id is required to link a profile`.
5. `user_id` empty after `TrimSpace` → `400`, description `user_id is required to link a profile`.
6. `GetProfile(profileId)` → `404` if the profile does not exist.
7. Already-linked handling — see below.
8. Otherwise `UpdateProfile(profileId, orgHandle, profileRequest)` writes the `user_id`.

> Steps 4 and 5 both reuse the **`UPDATE_PROFILE` error code**, not a link-specific one — only the free-text `Description` distinguishes "missing profile_id" from "missing user_id". Clients cannot branch on the code.

**Already-linked semantics** (`existingProfile.UserId != ""`):

| Case | Result |
|---|---|
| Linked to a **different** `user_id` | `409 Conflict`, new error `PROFILE_ALREADY_LINKED` |
| Linked to the **same** `user_id` | `200` with the same response body — **idempotent, no write performed** |

This is the write-once `user_id` rule from §9 enforced at the API edge for the first time. The re-link rejection came in as its own commit (`fa910df`, "Reject linking a profile that belongs to a different user"), separately from the endpoint itself (`77c2dc3`).

**New error code** in `internal/system/errors/error_codes.go`:

```go
PROFILE_ALREADY_LINKED = ErrorMessage{
    Code:        errorPrefix + "11017",
    Message:     "Profile already linked.",
    Description: "The profile is already linked to a different user",
}
```

**The update is a full rewrite, not a patch.** `LinkProfile` builds a `model.ProfileRequest` carrying the existing profile's `IdentityAttributes`, `Traits` and `ApplicationData` back in alongside the new `UserId`, then calls `UpdateProfile`. Application data is passed through `profileService.WideAppDataMap(...)`, which only **widens the static type** — `map[string]map[string]interface{}` (what `ProfileResponse.ApplicationData` is) to `map[string]interface{}` (what `ProfileRequest.ApplicationData` is). The body is a plain key-by-key copy; no reshaping happens, and the internal `[]ApplicationData` array form on the `Profile` model is not involved at any point. So the endpoint reads the whole profile and writes the whole profile back — there is a read-modify-write window here, and no optimistic concurrency check guarding it.

**After the link, unification proceeds as normal.** Setting `user_id` is what the unification worker watches for, so linking a profile whose `user_id` already has a permanent profile elsewhere results in a merge on the next pass — covered by the integration test `Link_UserId_With_Existing_Profile_Unifies`.

**Route shape changed mid-series.** The endpoint landed under a path-variable route and was moved to the flat collection route `POST /profiles/link` by `d2a78ed` before any release. Only the flat form exists at HEAD. Anything written against the earlier shape needs updating.

**Coverage** — `test/integration/profile_link_test.go` (326 lines, new): `Link_Anonymous_Profile_Sets_UserId`, `Link_With_Missing_Identifier_Is_Rejected`, `Link_Unknown_Profile_Is_Not_Found`, `Relink_To_Different_User_Is_Rejected`, `Relink_To_Same_User_Is_Accepted`, `Link_UserId_With_Existing_Profile_Unifies`. The suite also asserts that `profile:link` is present in `config/repository/conf/deployment.yaml` and maps to exactly `["internal_cds_profile_link"]`, so dropping the scope from config fails the build rather than silently opening the endpoint.

### Cookies

`profile_cookies` maps `cookie_id` (the `cds_profile` value) → `profile_id`, with `is_active`.
- Created when an anonymous profile is issued (`CreateProfileCookie`)
- Reused if a valid cookie is presented to `POST /profiles` (`handleExistingCookie` short-circuits creation)
- Deactivated on `SESSION_TERMINATE`
- Batch-deleted by the cleanup worker after the configured interval (default 24h, batch 500)

Cookie domain comes from `auth_server.cookieDomain`.

### Listing, pagination, filtering, projection

`GET /profiles` uses **cursor pagination**, not offset:
- `limit` via `pagination.ParsePageSize` (default `DefaultLimit = 50`)
- `cursor` is a base64-encoded `ProfileCursor { created_at, profile_id, direction }`; direction is `next` (older) or `prev` (newer)
- Response `pagination` block: `{ count, page_size, next_cursor, previous_cursor }`
- `filter` params, repeatable and also splittable on ` and `. Filter keys validated against `FilterRegex = ^[a-zA-Z0-9._-]+$`; schema filter fields limited to `attribute_name`, `application_identifier`
- `attributes` query param selects which fields to project (`parseRequestedAttributes` / `buildProfileListResponse`)
- **No consent filtering** on list — it is an administrative operation

### Deletion

`DELETE /profiles/{id}` soft-deletes: `delete_profile = true`, `list_profile = false`.

---

## 10. Profile Schema

### Effective schema and hierarchical sharing (Implemented; organization-management gated)

Primary-org identity attributes and mandatory identity consent are inherited in suborganizations. Non-complex `traits` and `application_data` attributes may be shared with all descendants or selected child/subtrees; identity attributes, complex or subattributes are not directly shareable. Read projections distinguish `OWNED`, `INHERITED` and `SHARED` and resolve active/conflicted resources. Local attributes take precedence; a shared duplicate that conflicts on name/application or owner becomes non-effective. Create/update/delete of a child-owned schema is rejected by child write guard. Policy changes and resource deletion affect computed effective availability; the source resource remains primary-owned.

**Interface:** `GetEffectiveProfileSchemaAttributes(ctx, orgHandle)` is used in profile/unification worker merges; attribute sharing handlers delegate policy CRUD to the sharing domain. Rule resolution must consult this effective model so a rule from an ancestor can refer to the correct visible attribute.


Defines the shape of data storable on a profile per organisation — names, types, mutability, and merge resolution.

### Scopes

| Scope | Description |
|---|---|
| `identity_attributes` | Sourced from IS claim dialects |
| `traits` | Behavioural / preference data |
| `application_data` | Per-application, namespaced by `application_identifier` |

Attribute names carry the scope prefix: `identity_attributes.email`, `traits.preferred_language`, `application_data.score`.

### Attribute model (`ProfileSchemaAttribute`)

| Field | Required | Description |
|---|---|---|
| `attribute_id` | auto | System-generated UUID |
| `attribute_name` | yes | Dot-notation path including scope prefix |
| `scope` | derived | |
| `display_name` | no | Human-readable label; max 50 chars, sanitised by `DisplayNameRegex`, camelCase auto-split |
| `value_type` | yes | See below |
| `merge_strategy` | yes | See below |
| `mutability` | yes | See below |
| `multi_valued` | no | Array of the declared type |
| `canonical_values` | no | `[{ value, label }]` |
| `sub_attributes` | no | `[{ attribute_id, attribute_name }]` when `value_type` is `complex` |
| `application_identifier` | no | Scopes the attribute to an application |
| `scim_dialect` | internal | Populated by schema sync; intended to be suppressed in responses |

### Value types

`string`, `integer`, `decimal`, `boolean`, `date`, `date_time`, `epoch`, `complex` — all present in `AllowedValueTypes`.

### Mutability

| Value | Meaning |
|---|---|
| `readWrite` | Freely read and updated |
| `readOnly` | System-managed — not updatable via API |
| `writeOnly` | Writable, not read back |
| `immutable` | Set at creation only |
| `writeOnce` | May start empty; once set, frozen |
| `computed` | **Declared as a constant but absent from `AllowedMutabilityValues`** — not accepted on writes |

### Merge strategies

| Strategy | Accepted on write? | Implemented in `MergeAttributeValue`? |
|---|---|---|
| `overwrite` | Yes | Yes — incoming replaces existing, unless incoming is empty |
| `combine` | Yes | Yes |
| `ignore` | **No** — rejected by `AllowedMergeStrategies` | Yes — keeps existing if non-nil |
| `latest` | No | No — falls through to default (returns incoming) |
| `oldest` | No | No |

`combine` semantics depend on `multi_valued`:
- `multi_valued: false` and both sides are maps → **deep merge** of the maps
- `multi_valued: true` → union by type: `string`/`text`, `integer`, `decimal`, `boolean`, `date_time` de-duplicate; `complex` **appends** without de-duplication

> `AllowedMergeStrategies` carries a `// todo: Remove later` against `overwrite`, so the accepted set is in flux.

### Core schema (built-in, not modifiable)

| Attribute | Type | Mutability |
|---|---|---|
| `profile_id` | `string` | `immutable` |
| `user_id` | `string` | `writeOnce` |
| `meta.created_at` | `date_time` | `readOnly` |
| `meta.updated_at` | `date_time` | `readOnly` |
| `meta.location` | `string` | `readOnly` |

### Write validation

`ValidateProfileAgainstSchema` enforces, per attribute: existence in schema, type correctness (incl. multi-valued and complex sub-attributes), canonical value membership, and mutability. Flattened sub-attribute keys are rejected outright (`rejectIfFlattenedSubAttribute`). Numeric equality is compared type-insensitively for mutability checks (`valuesEqualForMutability`).

---

## 11. Consent Management

### Current org-scoped consent constraints (Implemented)

Consent category get/list queries are explicitly filtered by org; requested category IDs not defined for the effective organization are excluded before consent-driven profile projection. Child organizations inherit the mandatory primary identity-data category rather than declaring a separate mandatory category. This remains consent/category authorization and filtering, **not** a new consent UX flow or external consent standard. Helm deployment still needs the consent-category OAuth scope mapping gap resolved separately.


Consent controls which profile attributes a caller may **read**. It does not gate writes.

### Key principles

| Principle | Detail |
|---|---|
| Category-level tracking | A profile consents to a whole category or not — no per-attribute granularity |
| Mandatory Identity Data | Every org has a built-in mandatory category, always enforced |
| System app bypass | System apps skip consent filtering entirely |
| Read-only gating | Applies to reads, not writes |
| No filtering on list | `GET /profiles` returns full profiles |

### Consent categories

An org-level definition declaring a name, a server-generated UUID `category_identifier`, a `purpose` (`profiling` | `personalization` | `destination`, per `AllowedConsentPurposes`), optional `destinations`, the covered attributes, and `is_mandatory`.

**Create/update request** (`ConsentCategoryRequest`):

```json
{
  "category_name": "Product Engagement",
  "purpose": "profiling",
  "destinations": ["segment"],
  "attributes": [
    { "attribute_name": "traits.engagement_score" },
    { "attribute_name": "traits.product_interests" },
    { "attribute_name": "application_data.events.event_name",
      "application_identifier": "i0QfDlYH6BIA8QygvmLRHikFrRIa" }
  ]
}
```

**Response** (`ConsentCategoryResponse`) adds `category_identifier`, `is_mandatory`, and a derived `scope` on each attribute. `org_handle` is deliberately omitted. `attribute_id` is internal and stripped by `ToResponse()`.

Validation:
- `category_identifier` always server-generated; caller values ignored
- Every `attribute_name` validated against the org's `profile_schema`; unknown → `400`
- `scope` derived from the name prefix, never supplied
- `application_data.*` attributes require `application_identifier`, and it must match the schema

**Error responses:**

| Status | Condition |
|---|---|
| 400 | Missing `category_name` or `purpose` |
| 400 | Invalid `purpose` |
| 400 | `attribute_name` not in the org's `profile_schema` |
| 400 | `applicationData` attribute missing or mismatched `application_identifier` |
| 403 | Attempt to modify or delete a mandatory category |
| 409 | Duplicate `category_name` |

### Attribute scopes (derived)

| Scope constant | Prefix | Example |
|---|---|---|
| `identityAttributes` | `identity_attributes.` | `identity_attributes.email` |
| `traits` | `traits.` | `traits.age` |
| `applicationData` | `application_data.` | `application_data.events.event_name` |

### applicationData matching quirk

`application_data` is a two-level map: `app_id → data_group_key → value`. `application_identifier` names the **outer** app ID, while `attribute_name` encodes the **inner** data-group key.

At filter time, matching is done on the **inner data-group key**, not the outer app ID. Consequence: if two apps both write under `"events"`, a single consent entry gates both.

### Complex attributes

Listing the parent `attribute_name` is sufficient — the whole object is returned. Sub-attributes need not be enumerated. Same for `applicationData`, since array entries cannot be partially filtered.

### Mandatory consent — Identity Data

- Auto-created when CDS is enabled; name `Identity Data`, purpose `profiling`, `is_mandatory: true`
- Covers all `identityAttributes`-scope fields
- Always applied — no `profile_consents` row needed, and it cannot be revoked
- Cannot be deleted or updated → `403`
- Stores **no rows** in `consent_category_attributes`; attributes are resolved live from `profile_schema WHERE scope = 'identity_attributes'` at filter time, so schema sync changes take effect automatically

### Per-profile consent records

```
profile_consents
  ├── profile_id
  ├── category_id     → consent_categories.category_identifier
  ├── consent_status  → true (consented) | false (revoked)
  └── consented_at
```
`UNIQUE (profile_id, category_id)`.

```
GET /t/{orgHandle}/cds/api/v1/profiles/{profileId}/consents
PUT /t/{orgHandle}/cds/api/v1/profiles/{profileId}/consents
```

`PUT` replaces the full non-mandatory consent set. Including a mandatory category → `403`.

Records are `ConsentRecord { category_identifier, is_consented, consented_at }`.

### Consent-scoped profile fetch

```
GET /t/{orgHandle}/cds/api/v1/profiles/{profileId}
GET .../profiles/{profileId}?consentCategoryId=<uuid>
GET .../profiles/{profileId}?consentCategoryId=<uuid1>&consentCategoryId=<uuid2>
```

`consentCategoryId` is parsed by `parseCommaSeparatedOrRepeated` — repeated params **and** comma-separated values both work.

| Caller | `consentCategoryId` | Response |
|---|---|---|
| System app | either | Full profile — filtering bypassed |
| Regular app | no | Mandatory identity data only |
| Regular app | yes, consented | Identity data + that category's attributes |
| Regular app | yes, not consented | Identity data only |
| Regular app | multiple | Identity data + **union** across consented categories |

Order of operations in `GetProfile`: fetch → `FilterApplicationData` (app scoping) → `FilterProfileByConsent` (only if not a system app).

### Data model

```
consent_categories
  ├── category_identifier  (UUID, server-generated, UNIQUE)
  ├── purpose              (profiling | personalization | destination)
  ├── is_mandatory
  └── consent_category_attributes
        ├── attribute_name
        ├── attribute_id   (FK → profile_schema.attribute_id ON DELETE CASCADE)
        ├── scope          (derived at write time)
        └── application_identifier

profile_consents
  ├── profile_id     (FK → profiles ON DELETE CASCADE)
  ├── category_id    (FK → consent_categories ON DELETE CASCADE)
  ├── consent_status
  └── consented_at
```

**Cascade behaviour:**

| Trigger | Effect |
|---|---|
| Delete `consent_categories` row | Removes its `consent_category_attributes` + `profile_consents` rows |
| Delete `profile_schema` attribute | Removes the `consent_category_attributes` row (category survives) |
| Delete `profiles` row | Removes all `profile_consents` for that profile |

---

## 12. Unification Rules

Rules now support two concurrent matching modes rather than solely exact comparison. This is **implemented**, not merely a proposed fuzzy roadmap.

### Rule structure and validation

| Field | Meaning |
|---|---|
| `rule_id`, `org_handle`, `rule_name` | UUID, owning organization, human-readable merge reason |
| `property_name`, `property_id` | Attribute path and schema foreign key; rule targets the effective schema where permitted |
| `priority`, `is_active` | Order and active state; smaller priority runs first |
| `attribute_type` | Comparator semantics, e.g. `PRIMITIVE_EXACT`, name, email, phone, DOB, `UNIQUE_ID` |
| `unification_method` | `deterministic` or `fuzzy`; effective method is coerced to exact when server-wide tolerant matching is off |
| `match_strength`, `mismatch_strength` | Nullable explicit `HIGH/MEDIUM/LOW` overrides; derived from type when missing; custom input gated by `allow_evidence_strength_override` |
| effective-only `origin`, `owner_org_handle`, `state` | Read projection for owned/shared rules and conflict/missing-attribute state, not extra SQL columns |

The legacy stored rule shape is backward-compatible: type defaults to `PRIMITIVE_EXACT`, method defaults deterministic and evidence strengths derive from type. Shared and owned rules are resolved as an **effective ordered list**: shared rules precede owned; shared resources and names are filtered for conflicts and missing compatible attributes; local rule property takes precedence when resolving collisions. The source of each shared rule remains the owning organization's row.

### Algorithm connection and lifecycle

`GetUnificationRules(ctx, orgHandle)` returns an effective set; `ActiveSortedByPriority` orders the viable rules for the identity-resolution worker. Each candidate is scored across the **entire applicable set**, not stopped at the first raw equality. The first rule whose verdict is `AGREE` supplies the principal score, while decisive exact rules and evidence objections affect routing; see §13. `UNKNOWN` placeholders and missing values are not negative evidence.

Activating a rule can backfill profile blocking keys for its attribute in batches; disabling/removing changes the index accordingly. `fuzzy_matching_enabled: false` writes exact keys instead of phonetic/LSH keys even during backfill, without rewriting the stored rule mode. Turning tolerant matching **on after it was off requires reindexing** (deactivate/reactivate relevant fuzzy rules), because historical fuzzy candidate keys were not written while disabled. See PR [#334](https://github.com/wso2/identity-customer-data-service/pull/334).

### Sharing policy API and permissions

`/unification-rules/{ruleId}/sharing-policies[/{policyId}]` supports owner-controlled policy CRUD. `GET` requires `unification_rules:share_view`; `POST/PUT/DELETE` require `unification_rules:share_update` as of PR [#355](https://github.com/wso2/identity-customer-data-service/pull/355). Child organizations cannot write rule and share-policy admin resources. `origin`, `owner_org_handle` may be present in responses for effective shared rules; custom evidence-strength fields are omitted unless the server exposes override controls.

### Operational cautions

`deterministic_match_decisive = true` preserves old exact-match merger outcomes (including decisive exact agreement outside the highest-priority rule); turning it off allows veto/conflicting evidence to hold more matches for review. A rule with low evidence strength cannot authorize an unattended merge merely because it is alone. A shared matchable value occurring on 50+ profiles can restrict unguarded auto-merge. Verify rules/thresholds with realistic data before lowering the default `auto_merge_threshold`.

---

## 13. How Profile Unification Works

### Current identity-resolution pipeline (Implemented)

**User/product intent:** link fragments belonging to one person, including spelling variations, without auto-merging two different people. Automatic unification is not treated as the only successful result: borderline cases are first-class review tasks.

1. **Changed matching evidence or indexed backfill.** A profile create/update/link or rule activation may trigger resolution; writes that do not change matchable values avoid redundant re-resolution. Existing user-ID matching and merge-lifecycle rules still apply.
2. **Bounded blocking candidate discovery.** Normalize configured rule values; generate exact, phonetic (Double Metaphone), locality-sensitive hash bands or phone suffix keys according to attribute type/mode. `blocking_keys` uses `(org_handle, attribute_name, key_value, profile_id)` as natural PK, 8 partitions on org handle then 4 on key value in PostgreSQL (32 leaf partitions), index on `(org_handle, profile_id)`. Candidate lookup groups keys by attribute/kind and limits results rather than scanning the full profile population. Stored keys must match enabled mode.
3. **Retrieve candidate profile state and evaluate typed rules.** Jaro-Winkler/parts-based names, normalized email, E.164-aware phone comparison, Jaccard for location where configured; deterministic rules use exact matching. `UNKNOWN` accounts for absent, sentinels, role addresses and otherwise nonidentifying values. Each applicable rule carries raw score, verdict and evidence strengths. `AGREE` means score ≥ review threshold; `DISAGREE` ≤ 0.3; other present comparisons may be `INCONCLUSIVE`.
4. **Priority waterfall, not average.** The first `AGREE` in rule run-order sets primary score. A perfect `UNIQUE_ID` match or an agreeing deterministic rule when `deterministic_match_decisive = true` returns final **1.0**. Otherwise, a `HIGH`-mismatch `DISAGREE`, a majority of other applicable contradictions, or weak/uncommon-value lone-agreement checks may cap final score to `auto_merge_threshold - 0.01`. The breakdown preserves **raw per-attribute** similarity, while final routing score includes caps.
5. **Decision by org settings.** With default `auto_merge_enabled = true`, `auto_merge_threshold = 1.0`, `manual_review_threshold = 0.75`, a final perfect score auto-merges; scores ≥0.75 but under 1.0 become review tasks; lower scores are unique/nonactionable in the write workflow. `auto_merge_enabled = false` disables unattended union. Child organizations inherit primary thresholds.
6. **Merge safety and human review.** Candidate and incoming profiles must be from the same org, and two permanent profiles with different user IDs are not mergeable. Failed/unmergeable auto-merge may fall back to review. `GET /identity-resolution/review-tasks` enumerates pending tasks; `POST /identity-resolution/review-tasks/{taskId}/resolve` approves or rejects with resolver metadata. Approval can change master/reference structure and is not automatically reversible. Review scopes split view vs update.
7. **Rejections survive ordinary edits.** The canonical unordered pair resides in `rejection_pairs` with score and raw breakdown seen at rejection. `ShouldReconsider` requires a materially higher score or a newly agreeing rule; ordinary unrelated edits do not reopen. A reconsidered rejection is **never auto-merged**: the worker reopens even an existing `REJECTED` task and asks the administrator again. SQLite string-vs-byte breakdown decoding is handled; see PR [#350](https://github.com/wso2/identity-customer-data-service/pull/350).

### Critical defaults, algorithm constraints and examples

- `DefaultAutoMergeThreshold = 1.0` since [#341](https://github.com/wso2/identity-customer-data-service/pull/341) (earlier `0.95`); `DefaultManualReviewThreshold = 0.75`; disagreement threshold `0.3`. **Perfect final score**, not identical raw text in every attribute, governs unattended routing.
- Example: same exact email with `deterministic_match_decisive` on → final 1.0 → eligible for automatic union despite other lower-priority concerns (legacy compatibility). An approximate 0.96 name alone with LOW match strength → capped below auto-merge, routed to review if above the review bar. Two different DOBs with HIGH mismatch strength can veto unattended merge unless a decisive exact rule short-circuits.
- A phone comparator evaluates normalized subscriber numbers and returns **1.0 or 0.0**, not a 0.9 score for matching last-seven-digit suffix; that suffix is for blocking candidate lookup only. Do not treat candidate lookup as matching evidence.
- `identity_resolution.fuzzy_matching_enabled = false` is a server gate across synchronous searches, async scoring and attribute backfill. Stored fuzzy rules remain intact; enabling later needs an index rebuild (activate each fuzzy rule again) to cover profiles indexed while off.
- There is **no guarantee of a quality/safety model score explanation endpoint** beyond stored review evidence, reason strings and per-attribute breakdowns. Review UX enhancements, richer explanations and audit insights may be product intent; distinguish them from implemented API data.

### Profile/master merge outcomes (carried forward)

The existing profile-domain merge routine determines the surviving master/reference structure for temporary↔permanent, temporary↔temporary, and same-user permanent↔permanent cases. Existing lineage flags and consent-aware profile representation remain; the identity-resolution engine supplies *candidate evidence and decision routing*, not a replacement profile storage model. Merge reasons use the decisive rule name where available rather than a generic tag. No standalone `/profiles/merge` endpoint is introduced by the delta.

### Tests and scale notes

Unit tests cover comparator normalization, phonetic/LSH key recall, no-value verdicts, score caps, evidence strength, priority, legacy exact behavior, rejected pair retention and reconsideration. SQLite and PostgreSQL integration suites exercise the real engine and isolation; PostgreSQL requires container/test infrastructure. The PR text reports measured synthetic blocking/index improvements but they are **not** externally verified production SLOs; instrumentation for candidate count, hit cap frequency and query cost remains desirable.

---

## 14. IS Sync — Identity Server Event Integration

CDS receives lifecycle events from WSO2 Identity Server via a webhook-style endpoint.

### Endpoint

```
POST /t/{orgHandle}/cds/api/v1/profiles/sync
Authorization: Basic <admin credentials>
```

### Request body (`ProfileSync`)

```json
{
  "event": "POST_ADD_USER",
  "userId": "…",
  "profileCookie": "…",
  "profileId": "…",
  "orgHandle": "carbon.super",
  "claims": { "…": "…" }
}
```

> `orgHandle` is read from the **body**, not the path, and is **required** — an empty value is a `400`. There is a `todo` in the source questioning this. The org must also be CDS-enabled or the event is rejected with `CDS_NOT_ENABLED`.

### Events

#### `POST_ADD_USER`
New user registered in IS.

**With cookie (anonymous → registered):** resolve the anonymous profile from the active cookie → attach `userId`, merge claims into `identity_attributes` → save; the unification worker promotes it to permanent.
**Without cookie:** if no profile exists for the `userId`, create a new permanent profile with the claims as `identity_attributes`.

#### `AUTHENTICATION_SUCCESS`
Fired on every successful login.

| Scenario | Action |
|---|---|
| Cookie + anonymous profile + a different existing permanent profile | Attach `userId` to the anonymous profile; enqueue; worker merges via `system:user_id_match` |
| Cookie + anonymous profile + no permanent profile | Promote the anonymous profile by attaching `userId` |
| Cookie already resolves to the user's own profile | No-op |
| No cookie | No-op (logged) |

#### `POST_SET_USER_CLAIM_VALUE_WITH_ID` / `POST_SET_USER_CLAIM_VALUES_WITH_ID`
Look up the permanent profile by `userId` → translate IS claim URIs to CDS attribute paths (`extractClaimKeyFromLocalURI`, `setNestedMapValue`) → merge into `identity_attributes` → save.

#### `SESSION_TERMINATE`
Look up the permanent profile by `userId` → resolve the cookie record → verify it belongs to that profile or one of its merged children → set `is_active = false`. Cleanup worker removes deactivated records in batches after the configured interval.

#### `POST_DELETE_USER_WITH_ID`
Look up by `userId` → soft-delete (`delete_profile = true`, `list_profile = false`).

---

## 15. Schema Sync

Keeps the CDS schema aligned with IS claim changes. Asynchronous via `SchemaSyncQueue` and `schema_sync_worker.go`.

### Endpoint

```
POST /t/{orgHandle}/cds/api/v1/profile-schema/sync
Authorization: Basic <admin credentials>
```

### Payload (`ProfileSchemaSync`)

```json
{
  "event": "POST_ADD_EXTERNAL_CLAIM",
  "orgHandle": "carbon.super"
}
```

> The Go struct field is `OrgId` but its **JSON tag is `orgHandle`** — the wire key is `orgHandle`.

### Events

| Event constant | IS trigger |
|---|---|
| `POST_ADD_EXTERNAL_CLAIM` | New SCIM claim added |
| `POST_UPDATE_EXTERNAL_CLAIM` | SCIM claim updated |
| `POST_DELETE_EXTERNAL_CLAIM` | SCIM claim deleted |
| `POST_UPDATE_LOCAL_CLAIM` | Local claim updated |
| `POST_DELETE_LOCAL_CLAIM` | Local claim deleted |

All events trigger the same action: a full re-sync of the org's schema from IS.

### How it works

1. IS fires the event to the CDS sync endpoint
2. CDS enqueues a `ProfileSchemaSync` job onto `SchemaSyncQueue`
3. The worker calls `SyncProfileSchema(orgId)`
4. Claim dialects are fetched from IS via the Identity Client (`/api/server/v1/claim-dialects`) and reconciled with the stored schema
5. New attributes added, changed ones updated, removed ones deleted; `scim_dialect` is recorded on each attribute

Best-effort — failures are logged but don't affect the HTTP response to IS.

### Scheduled sync

Beyond event-driven sync, `sync.schema` in `deployment.yaml` enables a periodic re-sync (`enabled: true`, `interval: 3600` seconds by default).

### Initial sync

On first enablement (`cds_enabled`), an initial sync bootstraps the schema from IS; `initial_schema_sync_done` is set on success.

Schema sync affects **definitions only** — it does not rewrite existing profile data, though updated merge strategies and mutability apply to all subsequent writes and unification runs.

---

## 16. Message Queue & Extending Providers

Two interfaces in `internal/system/queue/queue.go`:

| Interface | Purpose |
|---|---|
| `ProfileUnificationQueue` | `Enqueue(profileModel.Profile) error`, `Start(func(profileModel.Profile)) error`, `Close(ctx context.Context) error` |
| `SchemaSyncQueue` | `Enqueue(schemaModel.ProfileSchemaSync) error`, `Start(func(schemaModel.ProfileSchemaSync)) error`, `Close(ctx context.Context) error` |

`Start` must launch its consumer loop in a goroutine and return immediately. `Close` is idempotent **and deadline-aware**: it must return inside the supplied context. The ActiveMQ provider attempts a graceful STOMP disconnect, but if it cannot complete before the deadline it closes the underlying connection so shutdown does not leave a broker goroutine/socket behind.

### Built-in providers

| Constant | Value | Notes |
|---|---|---|
| `queue.TypeMemory` | `"memory"` | Default; `internal/system/queue/inmemory`; `DefaultQueueSize = 1000`; single-instance only |
| `queue.TypeActiveMQ` | `"activemq"` | `internal/system/queue/activemq`, STOMP; durable, for multi-instance |

ActiveMQ locally:

```bash
docker run -d -p 61613:61613 --name activemq \
  -e ACTIVEMQ_ADMIN_LOGIN=admin -e ACTIVEMQ_ADMIN_PASSWORD=admin \
  rmohr/activemq
```

```yaml
message_queue:
  type: "activemq"
  broker:
    addr: "localhost:61613"
    username: "admin"
    password: "${BROKER_PASSWORD}"
    profile_queue_name: "/queue/cds-profile-unification"
    schema_sync_queue_name: "/queue/cds-schema-sync"
```

### Adding a provider

1. Create `internal/system/queue/myprovider/myprovider.go`
2. Implement both interfaces (3 methods each); `Close(ctx)` must honor cancellation/deadlines and be safe to call repeatedly
3. Register via `init()` using `queue.RegisterProfileQueueProvider` / `queue.RegisterSchemaSyncQueueProvider` (factory in `queue/factory.go`)
4. Blank-import in `cmd/server/main.go` — as ActiveMQ already is
5. Set `message_queue.type: "myprovider"` in `deployment.yaml`
6. Optionally mirror `test/activemq_integration/`

```go
func init() {
    queue.RegisterProfileQueueProvider("myprovider",
        func(cfg config.ExternalBrokerConfig, tlsCfg config.TLSConfig) (queue.ProfileUnificationQueue, error) {
            return newProfileQueue(...)
        },
    )
    queue.RegisterSchemaSyncQueueProvider("myprovider",
        func(cfg config.ExternalBrokerConfig, tlsCfg config.TLSConfig) (queue.SchemaSyncQueue, error) {
            return newSchemaSyncQueue(...)
        },
    )
}
```

`ExternalBrokerConfig` carries `addr`, `username`, `password`, `profile_queue_name`, `schema_sync_queue_name`. Anything provider-specific goes in env vars or a new sub-section.

---

## 17. Database Schema

### New current SQL schema and migrations (Implemented)

- `dbscripts/migrations/001-identity-resolution-postgres.sql`: additive unification-rule typed/evidence fields plus `blocking_keys`, `review_tasks`, `rejection_pairs`, associated indexes and partitions. The `postgres.sql` fresh-install schema contains the new tables. **Do not assume a `merge_audit_log` table exists** merely from the PR prose: current inspected fresh-install SQL did not define it.
- `dbscripts/migrations/002-organization-management-postgres.sql`: additive `organizations`, `resource_sharing_policy`, `resource_sharing_policy_target` and indexes. **The actual DDL uses `resource_sharing_policy*`**, not draft prose's `cds_share_policy*`.
- SQLite current-install script and test setup schemas have matching functional tables, with SQL-dialect differences (JSONB/TEXT, partitioning). `test/schema/{schema_parity,migration_parity}_test.go` aims to detect schema drift.
- **Deployment rule:** existing databases should run the additive migration scripts *before* enabling new features; fresh database bootstrap uses current complete scripts. No automatic backfill of fuzzy keys occurs merely because the switch was turned on. PostgreSQL 8×4 partition shape should be treated as deliberately measured, not casually changed during upgrades.


Two scripts ship, defining the same 13 tables:

- `dbscripts/postgres.sql` — applied by the operator (`psql < ...`). Uses `CREATE TABLE` (no `IF NOT EXISTS`), `TIMESTAMPTZ ... DEFAULT now()`, `TEXT[]`, `jsonb` defaults, and creates the `pg_trgm` extension plus GIN indexes.
- `dbscripts/sqlite.sql` — embedded in the binary (`dbscripts/embed.go`, `//go:embed`) and applied automatically on first start. Idempotent (`CREATE TABLE IF NOT EXISTS`), so re-applying is a no-op.

The table/column list below is common to both.

| Table | Purpose |
|---|---|
| `profiles` | `profile_id` PK, `user_id`, `org_handle`, `created_at`, `updated_at`, `location`, `origin_country`, `list_profile`, `delete_profile`, `traits` JSONB, `identity_attributes` JSONB |
| `profile_reference` | `profile_id` PK, `org_handle`, `profile_status`, `reference_profile_id`, `reference_profile_org_handle`, `reference_reason` — the merge lineage |
| `application_data` | `app_data_id` PK, `profile_id` FK CASCADE, `app_id`, `application_data` JSONB, `UNIQUE(profile_id, app_id)` |
| `applications` | `app_id` PK, `org_handle`, `client_id`, timestamps, `UNIQUE(org_handle, client_id)` |
| `profile_schema` | `attribute_id` PK, `scope`, `org_handle`, `attribute_name`, `display_name`, `value_type`, `merge_strategy`, `application_identifier`, `mutability`, `multi_valued`, `canonical_values` JSONB, `sub_attributes` JSONB, `scim_dialect` |
| `unification_rules` | `rule_id` PK, `org_handle`, `rule_name`, `property_name`, `property_id` FK → `profile_schema` CASCADE, `priority`, `is_active`, timestamps |
| `consent_categories` | `id` PK, `org_handle`, `category_name`, `category_identifier` UNIQUE, `purpose`, `destinations` TEXT[], `is_mandatory` |
| `consent_category_attributes` | `category_id` FK CASCADE, `scope`, `attribute_name`, `attribute_id` FK → `profile_schema` CASCADE, `application_identifier`, `UNIQUE(category_id, scope, attribute_name, application_identifier)` |
| `profile_consents` | `profile_id` FK CASCADE, `category_id` FK CASCADE, `consent_status`, `consented_at`, `UNIQUE(profile_id, category_id)` |
| `profile_cookies` | `cookie_id` PK, `profile_id` FK CASCADE, `is_active` |
| `cds_config` | `(org_handle, config)` PK, `value` — backs admin config |
| `profile_unification_modes` | `org_handle`, `merge_type`, `rule`, `UNIQUE(org_handle, merge_type, rule)` — for the planned approval workflow |
| `profile_unification_triggers` | `org_handle` UNIQUE, `trigger_type`, `last_trigger`, `duration` — for scheduled unification |

**Notable indexing (PostgreSQL):** composite `(org_handle, created_at, profile_id)` supporting cursor pagination; `(org_handle, user_id)` (also used by the bounded same-user unification lookup); `pg_trgm` GIN on `user_id` for contains-matching; GIN on `profiles.traits` and `profiles.identity_attributes` (the new rule-candidate JSONPath prefilter is tested for GIN eligibility); GIN on `application_data->'app_specific_data'`; `(org_handle, is_active, priority)` on unification rules.

> `CREATE EXTENSION IF NOT EXISTS pg_trgm` means the DB user needs extension-creation rights on first init.

**Indexing on the inbuilt database:** the B-tree indexes carry over (`idx_profiles_org_created_profile`, `idx_profiles_org_user`, the three `profile_reference` indexes, both `application_data` indexes, both `profile_schema` indexes, both `unification_rules` indexes). The four GIN/trigram indexes have **no equivalent** — SQLite has neither `pg_trgm` nor GIN.

### Inbuilt-database type & behaviour differences

| Aspect | PostgreSQL | Inbuilt SQLite |
|---|---|---|
| Timestamps | `TIMESTAMPTZ DEFAULT now()` | `TIMESTAMP DEFAULT (strftime('%Y-%m-%d %H:%M:%f','now') \|\| '+00:00')` — sortable UTC text, which keyset pagination orders on |
| JSON columns | native `jsonb` | `JSONB` declared type over TEXT storage; read via `json_extract` |
| Booleans | native | stored as `0`/`1`, coerced back in `client/normalize.go` |
| `destinations` | `TEXT[]` | JSON array text — written by `scripts.EncodeStringArray(dbType, …)` (replaced the direct `pq.Array` calls), read by `DecodeStringArray`, which accepts either form |
| `VARCHAR(n)` | length enforced | **not enforced** |
| `co` / `sw` profile filters | trigram-accelerated | correct but **full table scan** |
| Case-insensitive match | `ILIKE`, locale-aware | `LIKE`, ASCII only |
| Concurrency | multi-instance | single file, single process — **not shared storage** |

**SQLite DSN defaults** (`database.DefaultSQLiteOptions`, all required):
`_pragma=foreign_keys(1)` (enforces the schema's `ON DELETE CASCADE`), `_pragma=journal_mode(WAL)` (readers alongside a writer), `_pragma=busy_timeout(5000)`, `_txlock=immediate` (take the write lock at transaction start), `_time_format=sqlite`, `_timezone=UTC`, `_texttotime=true`. Setting `datasource.sqlite.options` **replaces** the whole string, so any override must restate every option it still needs.

Default pool size is 4 (`DefaultSQLiteMaxOpenConns`) — SQLite serialises writers, so a small pool avoids lock contention.

> **Fixed in v0.3.64:** `DeleteInactiveCookies` (`CDS-CKI-08`) targeted a non-existent `cookie_profiles` table — the table is `profile_cookies`, as every other cookie statement uses. `cookie_cleanup_worker.go` logs the failure via `logger.Debug`, so it failed silently once per `cleanup.cookie.interval` (86400s) in every deployment since the statement was introduced; inactive cookie rows have been accumulating. **Every statement in `AllQueries()` is now prepared against both datasources in tests**, which is what surfaced it.

---

## 18. Deployment & Configuration

### Latest deployment state (`3178ceb`; Implemented / Operational caveats)

| Surface | Current behavior |
|---|---|
| Product version | `version.txt = v0.6.1` (release automation; main commit is separate identifier) |
| Helm chart | `version: 0.0.38`, `appVersion: "0.0.1"` placeholder; do not treat as matching product version |
| Go toolchain | `go.mod`, `Dockerfile`, PR CI and release workflow pinned to **Go 1.26.10**, fixing a reported Go stdlib `crypto/x509` chain verification issue; pre-existing published images require **rebuild and re-scan**, not just a source merge |
| Tolerant matching | `identity_resolution.fuzzy_matching_enabled: false` default; Helm values `cloud.deployment.cds.config.identityResolution.fuzzyMatchingEnabled`; moving off→on requires reindex |
| Evidence override | `identity_resolution.allow_evidence_strength_override: false` default; Helm values `cloud.deployment.cds.config.identityResolution.allowEvidenceStrengthOverride` |
| Child organizations | `organization_management.enabled: false` default; chart `cloud.deployment.cds.config.organizationManagement.enabled` (check exact rendered values/template spelling before automation) |
| Database upgrades | Apply `001-identity-resolution-postgres.sql` and `002-organization-management-postgres.sql` as appropriate before feature enablement; fresh scripts already include tables |
| OAuth configuration | Include review view/update and sharing policy share_view/share_update in both IS scopes and CDS required scopes; child org scopes are distinct; chart consent-category scopes remain a known gap |

**Feature rollout sequence:** migrate DB → deploy Go 1.26.10 rebuild → install/render new chart/config → ensure IS scopes and CDS API resource registration → enable `organization_management` (if needed) → synchronize organization tree and wait for `SYNCED` → set child access policies and optional resource shares → explicitly opt into fuzzy matching → rebuild affected keys and validate review queue, rejection and org-isolation flows. Use a staged/test environment; GitHub merges are not evidence of successful live deployment.

**Release-note workflow since #335:** the type-labeling workflow no longer turns missing PR type into a failing check. It copies type labels from linked issues where possible, otherwise defaults to `Type/Bug` with a warning; `continue-on-error` prevents a labeling inconvenience from blocking CI. Events are narrowed to open/reopen/edit/label/unlabel. `.github/release.yml` still groups generated release notes and retains an Other Changes fallback. This is repository automation, not CDS API behavior.

**Known tests/ops gaps:** no committed Helm template render test identified for the new identity-resolution configuration; PR #338 validated template structure by reading rather than running Helm. PR #350 authors reported SQLite integration tests but were unable to run PostgreSQL locally due to Docker storage; verify current CI (and migration parity) before live upgrade. Instrument candidate-set sizes, group cap hits, index backfill failures, org-tree sync and review latency for production operation.


### Datasource choice

`datasource.type` selects the store. **An unset value means `sqlite`** (`database.DefaultType`), though the shipped `deployment.yaml` sets `postgres` explicitly, so switching requires editing it.

| | Inbuilt (`sqlite`) | PostgreSQL (`postgres`) |
|---|---|---|
| Setup | none — file + schema created on first start | run a server, apply `dbscripts/postgres.sql` |
| Docker required | no | yes (or an existing server) |
| Intended for | local dev, demos, evaluation, single-instance | production, multi-instance |

Use PostgreSQL when you need: more than one CDS instance on the same data (the Helm chart defaults to **2 replicas**, so it requires PostgreSQL); trigram-accelerated `co`/`sw` filters; non-ASCII case-insensitive matching; or `VARCHAR(n)` length enforcement.

### Local run — Option A: inbuilt database (no Docker)

```yaml
# repository/conf/deployment.yaml
datasource:
  type: "sqlite"
```

```bash
make all
cd target && unzip cds-*.zip && cd cds-* && ./cds
```

The server creates `repository/database/cds.db` (relative to CDS home) and applies the embedded schema. Override the location with `datasource.sqlite.path` — a relative path resolves against `CDS_HOME`, an absolute path is used as-is. `.gitignore` now excludes `config/repository/database/`, `*.db`, `*.db-wal`, `*.db-shm`.

### Local run — Option B: PostgreSQL

```bash
docker run -d -p 5432:5432 --name postgres \
  -e POSTGRES_USER=cdsuser -e POSTGRES_PASSWORD=cdspwd -e POSTGRES_DB=cdsdb postgres

docker exec -i postgres psql -U cdsuser -d cdsdb < dbscripts/postgres.sql

make all
cd target && unzip cds-*.zip && cd cds-* && ./cds
```

Set `DB_PASSWORD=cdspwd` in `dev.env`. The server **refuses to start** if `type: postgres` and any of `hostname`, `username`, `password`, `name` is empty (`provider.ValidateDataSource`). At first pool creation it also verifies PostgreSQL with `PingContext`; connection establishment is bounded by `datasource.postgres.connect_timeout_seconds` (default 10s), so an unreachable database fails startup within a known bound rather than relying on an OS-level timeout.

Optional PostgreSQL pool settings live under `datasource.postgres`: `max_open_conns` (100), `max_idle_conns` (25), `conn_max_lifetime_seconds` (1800), `conn_max_idle_time_seconds` (300), `connect_timeout_seconds` (10). These values are per CDS process/pod.

> The README's PostgreSQL snippet and script filename were corrected upstream in v0.3.63; the note in the previous version of this context about that typo no longer applies.

### Local run — Option C: scripted setup (`scripts/local-setup/script.sh`) — new in v0.3.65

Options A and B start CDS alone. **A stock IS pack has no CDS configuration**, so a CDS started that way authenticates but receives no user events. `scripts/local-setup/script.sh` does the full wiring: builds CDS and the IS extension bundles, generates and exchanges the TLS certificates, writes both configurations, registers the OAuth applications, enables CDS for the organization, starts both servers, and runs smoke tests in both directions.

```bash
./scripts/local-setup/script.sh up                 # inbuilt SQLite, no external dependencies
./scripts/local-setup/script.sh up --db postgres   # CDS on PostgreSQL in a Docker container

./scripts/local-setup/script.sh start              # ~20s — reuses an existing provisioned setup
./scripts/local-setup/script.sh restart
./scripts/local-setup/script.sh down [--purge]
./scripts/local-setup/script.sh status
./scripts/local-setup/script.sh logs cds|is
```

`up` is idempotent and provisions; `start` builds nothing, clones nothing, runs no Maven and calls no management API. `up` records the datasource, ports, offset and tenant in the work directory, and `start`/`restart`/`down`/`status`/`logs` reuse them — command-line options still win. Re-run `up` to pick up a CDS code change: it rebuilds the binary, restarts CDS and leaves IS alone.

**Requirements:** `java` 21, `maven`, `go`, `jq`; `docker` only for `--db postgres`; plus `curl`, `git`, `unzip`, `openssl`, `lsof`, `keytool`. `product-is` builds with source and target 21, so an older JDK cannot compile it — the script prefers an installed JDK 21 over the machine default and stops if the resolved one is older; an exported `JAVA_HOME` wins.

**Work directory.** Everything lives in `.local-dev/` next to the repo (`--work-dir PATH` to move it); the repository itself is never modified, and `.gitignore` now excludes `.local-dev/`.

```text
<work-dir>/
  src/product-is/                          IS pack source checkout
  src/identity-customer-data-service-extensions/
  is/wso2is-<version>/                     extracted pack, used as IS_HOME
  bin/cds                                  the CDS binary
  cds-home/repository/conf/deployment.yaml generated CDS configuration
  cds-home/repository/database/cds.db      inbuilt database (--db sqlite)
  cds-home/etc/certs/                      CDS key pair + the IS certificate
  logs/{cds,is,is-build}.log
  run/{cds,is}.pid
  state.env                                client IDs, secrets, last-`up` settings — mode 0600
```

`state.env` holds the client secrets and lives outside the repository. The generated `deployment.yaml` carries **no** secrets: its `${NAME}` placeholders are left in place and CDS expands them from the environment at load time, which the script exports when it starts the binary.

**The IS pack.** The Console renders the **Customer Data** section only if the IS build knows the `cds_host` key, so the pack must be recent. By default the script shallow-clones [`wso2/product-is`](https://github.com/wso2/product-is) and runs `mvn clean install -Dmaven.test.skip=true`. First build 10–30 min; warm rebuild ~1 min; later runs reuse the extracted pack. `--is-zip PATH` skips the build, `--is-src PATH` builds an existing checkout, `--is-ref REF` picks the branch/tag (default `master`), `--is-rebuild` forces a rebuild, `--clean` re-extracts. Only `github.com` and the public WSO2 Maven repository are contacted.

**Ports.** CDS HTTPS `8900` (`--cds-port`); IS `9443`/`9763` (`--is-offset N` shifts both); PostgreSQL `5432` (`--pg-port`). `up` fails if a port is taken; `--force` kills the holder. A second concurrent setup needs its own `--work-dir`, `--is-offset`, `--cds-port` and `--pg-container`.

**What it configures** — each item is a thing CDS silently does not work without:

| | Why it matters |
|---|---|
| Four `org.wso2.identity.customer.data.service.*` bundles in `repository/components/dropins`, **with** their `[[event_handler]]` subscriptions | `AbstractEventHandler.canHandle()` returns false without a module config — a deployed bundle receives no events |
| CDS API resources and scopes, seeded via `deployment.toml` | IS reserves the `internal_` prefix and rejects `internal_cds_*` over REST |
| `[console.extensions] cds_host` | Makes the Console call CDS directly rather than building CDS URLs off the IS origin; those cross-origin calls rely on CDS `auth.cors_allowed_origins` |
| Certificate exchange both ways (CDS cert → IS `client-truststore.p12`, IS cert → CDS trust store) | CDS does not start with an unreadable `tls.trust_store`, and neither self-signed cert is in the system roots |
| `http://wso2.org/claims/cdsProfile` local claim + SCIM2 mapping | Without it, user events carry no profile cookie (§9 Cookies, §14) |
| Relaxed policy on `/oauth2/introspect` | Lets CDS introspect with its own client credentials instead of admin-user Basic auth |
| Two OAuth apps — system app (CDS→IS) and client app with `aud=iam-cds` (caller→CDS) | The Console is separate because it issues opaque tokens |
| `cds_enabled` for the organization | Every profile, schema and rule endpoint returns `400` until set (§6); setting it also runs the initial profile-schema sync and seeds the default consent category |

**Templates over script edits.** `script.sh` handles arguments, ordering and process control; the generated configuration lives in `templates/`, one file per artifact — `is-deployment.toml` (appended to the pack's `deployment.toml` between `# BEGIN/END cds-local-dev` markers, so a re-run replaces it), `cds-deployment.yaml` (copied to the CDS home, then the `required_scopes` block read out of `config/repository/conf/deployment.yaml` and one `datasource-*.yaml` fragment appended), `datasource-sqlite.yaml` / `datasource-postgres.yaml`, `openssl.cnf`. Change behaviour by editing a template, not the script. `is-deployment.toml` and `openssl.cnf` use `${NAME}` placeholders the script substitutes (an unfilled placeholder is an error); the CDS template's placeholders are deliberately left for CDS to expand from the environment.

> Because the CDS template is a **full configuration rather than an overlay**, a section added upstream to `config/repository/conf/deployment.yaml` would be silently missed. `up` compares the two and warns. Worth remembering whenever §18's `deployment.yaml` table gains a row.

**Smoke tests** (`up` runs them; `--skip-tests` to skip, `--tests` to add them to `start`):

1. `GET /cds/api/v1/ready` → 200 — CDS up, database reachable
2. a client-credentials token carries `aud=iam-cds`
3. the same token carries the expected `org_handle`
4. CDS reports the organization as enabled
5. `GET /profiles` → 200 — introspection and scope mapping work
6. `GET /profile-schema` → 200 — CDS reached the IS claim APIs
7. a user created in IS appears as a profile in CDS — dropins, event handlers and the IS→CDS sync path work

Test 7 creates and deletes an IS user; `--keep-test-user` keeps it.

**Scope limits.** Super tenant `carbon.super` only (`--tenant` exists but no other value is supported). Only the CDS datasource is configurable — IS keeps its default H2 databases. CDS runs with the in-memory queue (§16 for ActiveMQ). Certificates are self-signed and IS keeps its default administrator, so clients talk to both servers with verification off. This is a development setup, not a deployment reference.

Documented at `docs/guides/local-development.md`, linked from `docs/README.md`.

### `deployment.yaml` sections

| Section | Notable keys |
|---|---|
| `env`, `log` | `${ENV}`, `${LOG_LEVEL}` |
| `cache` | `enabled`, `default_cache_timeout: 15` |
| `auth` | `cors_allowed_origins` |
| `addr` | `host: 127.0.0.1`, `port: 8900` |
| `server_url` | Public base URL used to construct `Location` headers |
| `auth_server` | host/port, IS endpoints, `client_id`/`client_secret`, admin credentials, `introspection_client_id`/`_secret`, `isSystemAppGrantEnabled`, `cookieDomain`, `required_scopes` |
| `application_identifier_type` | `client_id` (default) or `app_id` |
| `sync.schema` | `enabled`, `interval` (3600s) |
| `cleanup.cookie` | `enabled`, `interval` (86400s), `batch_size` (500) |
| `shutdown` | `grace_period_seconds` — bounds HTTP shutdown + all workers + broker close before the DB pool closes; default/shipped value `25`, negative values rejected at startup |
| `message_queue` | `type` + `broker` block |
| `datasource` | `type`: `sqlite` (default when unset) or `postgres`. PostgreSQL base keys — host/port/name/user/password, `sslmode`; pool keys under `datasource.postgres` — `max_open_conns` (100), `max_idle_conns` (25), `conn_max_lifetime_seconds` (1800), `conn_max_idle_time_seconds` (300), `connect_timeout_seconds` (10). SQLite keys under `datasource.sqlite` — `path` (default `repository/database/cds.db`), `options` (DSN query string; replaces defaults wholesale), `max_open_conns` (default 4). |
| `tls` | `mtls_enabled`, `cert_dir`, `server_cert`, `server_key`, `client_cert`, `trust_store` |

Secrets are `${ENV_VAR}`-interpolated: `AUTH_SERVER_CLIENT_SECRET`, `AUTH_SERVER_ADMIN_PASSWORD`, `INTROSPECTION_CLIENT_SECRET`, `BROKER_PASSWORD`, `DB_PASSWORD`.

### Helm (`install/helm`)

Templates for deployment, service, config, env secret, HPA, PDB, and **two ingresses** — `cds-health-ingress` (non-tenanted `/cds/...`) and `cds-tenant-ingress` (`/t/{org}/...`), matching the routing split in §4. `values.yaml` sets `replicas: 2`, so a Helm deployment requires **both** PostgreSQL (the inbuilt database is not shared storage) and the ActiveMQ queue rather than the in-memory default.

Current runtime-related Helm defaults: PostgreSQL pool `100/25`, lifetime `1800s`, idle `300s`, connect timeout `10s`; readiness `timeoutSeconds: 5`; application shutdown grace `25s`; pod `terminationGracePeriodSeconds: 30`. The chart source version at `a9fdadd` is **`0.0.34`** (`appVersion` is still the placeholder `"0.0.1"`).

### Versioning & release process — reworked (new `docs/process/releasing.md`)

`.github/workflows/version-bump.yml` (151 lines) was **deleted**, and `.github/workflows/release.yml` absorbed its job. The semantic change matters for anyone reading `version.txt` to find out what `main` contains:

| | Before (≤ `a9bad2f`) | After (`f4b1da6`) |
|---|---|---|
| Who owns `version.txt` | the version-bump workflow | the release builder |
| When it moves | on merge to `main` | only when a release is cut |
| What it means | roughly "current `main`" | **the last released version** |
| Release dispatch input | `version`, required, must equal `version.txt` | `branch`, `version`, `bump_type`, `use_existing_version`, `commit_version_bump`, `prerelease` — all optional |

Version resolution precedence in the release workflow: explicit `version` input wins; else `use_existing_version: true` releases `version.txt` as-is with nothing committed; else `bump_type` is applied to `version.txt` (`patch` → z+1, `minor` → y+1.0, `major` → x+1.0.0).

> **The default bump type is `minor`** (`fa454ef`), and `minor` is also listed first in the choice options so it is what the dispatch UI preselects. The intended split, per the workflow's own header comment: a **scheduled** release is a `minor` bump off `main`; a **maintenance** release is a `patch` bump off the relevant release branch. That predicted transition has now happened: **`v0.4.0` was published on 2026-10-13**, followed by the **`v0.4.1` patch release on 2026-10-14**. This is a release event, not a new profile/API capability.

Other release-path changes:

- A release is **refused if the resolved tag already exists** (`git rev-parse "$VERSION"`) — the tag, not the file, is now the release identity.
- The version regex accepts an optional prerelease suffix: `^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9a-zA-Z.-]+)?$`.
- Releases can be built from **any branch** via the `branch` input, defaulting to `main` — this is what makes maintenance releases off a release branch possible.
- `version.txt` is always *written* so `make build` names the artifact correctly, but only *committed* when `commit_version_bump` is true.
- The workflow checks out with `secrets.CDS_RELEASE_PAT || secrets.GITHUB_TOKEN`; commits are authored as `wso2-iam-bot`.
- `update-deployment-versions.yml` no longer looks up the tag by the dispatched commit SHA — because the builder now commits the bump *and then* tags it, the dispatched SHA is not the tagged one. It resolves `git describe --tags --abbrev=0 --match 'v[0-9]*'` against the released branch tip instead.

Helm chart version has since advanced again: `0.0.32` (`b38e9bd`) → `0.0.33` after PR #309 → **`0.0.34` at `71fb3bd`** after PR #312. The chart version is tracked separately from the product version; `appVersion` remains the stale placeholder `"0.0.1"`.

### October 2026: v0.4.0 / v0.4.1 and build/runtime maintenance (new since `a73cb1b`)

- **Release state.** Release workflow commits by `wso2-iam-bot` changed `version.txt` from `v0.3.65` to `v0.4.0` (2026-10-13), then to `v0.4.1` (2026-10-14). Both releases are published on GitHub. The `v0.4.0` release packages previously merged features; it should **not** be interpreted as introducing those changes during this reporting window. The `v0.4.1` release notes list [PR #325](https://github.com/wso2/identity-customer-data-service/pull/325) as the new code change.
- **Go toolchain patch alignment.** [PR #325](https://github.com/wso2/identity-customer-data-service/pull/325), author `SujanSanjula96` (no assignee), changes `go.mod`'s `go` directive, the `GO_VERSION` value in `.github/workflows/pr-builder.yml`, the `actions/setup-go` version in `.github/workflows/release.yml`, and `Dockerfile`'s `GO_BASE` from `1.26.5` to `1.26.9`. This aligns local declared minimum, CI, release and Docker builds; it does **not** change the `require` module dependency versions or `go.sum` in this diff.
- **Container OS packages.** The `Dockerfile` Alpine runtime stage now runs `apk upgrade --no-cache` before creating the non-root application user. It upgrades packages present in the chosen base image at **image build time**, not when CDS starts. `RUNTIME_BASE` still defaults to the mutable `alpine:latest` tag, so reproducible builds require control over base-image digests/package provenance. No changed Helm values or deployment config keys accompany this update.
- **Compatibility/verification.** No Go application source files, SQL scripts, API definitions, schema migrations, new tests or Helm chart templates changed in `a73cb1b..49c5abc`. Existing CI is configured for Go `1.26.9`; this code diff alone does not establish that a production image has been redeployed or that all existing tests passed. The release artifact/tag should be validated in deployment pipelines as usual.


`docs/process/releasing.md` (134 lines, new) documents the model, inputs, scheduled/maintenance/major/prerelease runs, retrying a failed release, post-release steps, verification, Helm chart versioning and troubleshooting. `docs/README.md` gained a **Process** table pointing at it. Commit `3c56a87` ("Reword the release doc to avoid cutting a release") rewrote instruction text that, followed literally, would have triggered a real release.

### PR classification & generated release notes — current

> **Repository automation, not runtime.** The existing release-note categories and `skip-changelog` editorial exclusion remain. PR [#335](https://github.com/wso2/identity-customer-data-service/pull/335), owner `VivekVinushanth`, replaces failing missing-type checks with best-effort type propagation/default (`Type/Bug`) and warnings. The workflow runs as `pull_request_target` but does not check out or execute untrusted PR code; `pull-requests: write` is limited to label updates and the job uses `continue-on-error: true`.

Unlike the 2026-10-26 snapshot, there is no separate check job that automatically fails a PR for missing type and no `synchronize` / `ready_for_review` trigger. Manual label correction and the release-note `Other Changes` fallback remain available. Release builder versions advanced through product `v0.6.1`; chart `0.0.38` is independent. This workflow does not change application runtime, API, SQL, container behavior or Helm feature flags.

### Tests

`test/integration`, `test/activemq_integration`, `test/setup`, plus `Integration_Test_Guide.MD` and `Unification_Scenarios.MD`.

The integration suite runs against either datasource: `CDS_TEST_DB=sqlite` selects the inbuilt database (`test/setup/sqlite_test_db.go`), anything else spins up PostgreSQL via testcontainers. Added in v0.3.63–64: `test/integration/profile_pagination_test.go`, `test/integration/queries_prepare_test.go` (prepares every statement in `scripts.AllQueries()` — this is what would have caught the `cookie_profiles` bug), `test/setup/query_prepare.go`. `TEST_MODE=true` as the signal for "don't close the pool" was replaced by the `shared` flag on `DBClient`. Added at `b38e9bd`: `test/integration/profile_link_test.go` (§9), which is also the first integration test to assert on the **contents of `deployment.yaml`** rather than on API behaviour alone.

CI (`.github/workflows/pr-builder.yml`) has a `sqlite-test` job alongside the PostgreSQL one. At the **2026-10-26 historical baseline**, PR CI and release build Go setup both used **Go 1.26.9**, consistent with `go.mod` and the Docker builder; this interval introduced no new Go test files. PR #330 describes eight mock validation scenarios for its label workflow, a GraphQL query check and a generated-notes dry run; those are PR-author verification claims rather than committed repository test files or proof of successful live workflow execution. `make unit-test` runs `go test -race` across the internal packages because the process shares database pools and worker state concurrently. The `b38e9bd` → `71fb3bd` range added coverage for pool ownership/concurrent initialization/config validation/connect timeout, context cancellation and request deadlines, transaction context behaviour, production-provider lifecycle, ActiveMQ close deadlines, worker draining and server shutdown ordering.

The `71fb3bd` → `a73cb1b` range adds `internal/system/workers/profile_worker_test.go` and `test/integration/unification_candidates_test.go`. Coverage includes user-ID precedence, rule priority, single-merge behaviour, conflicting user IDs, queued child/waiting-profile handling, application-data preservation/failure paths, organization/reference-profile scope, deterministic oldest-candidate ordering, context cancellation, invalid property paths, exact cross-datasource string/array semantics, and PostgreSQL GIN eligibility. The integration test also contains a PostgreSQL lookup benchmark for **1,000 / 100,000 / 1,000,000** master profiles across user-ID matches, selective rule matches, misses and a common attribute value. Those numbers are lookup-only synthetic measurements, not full-merge or production-throughput guarantees.

`dependency-validation.yml` remains hardened to pin the PR head SHA and fail if the head moved mid-run.

> Latent bug in `test/integration/main_test.go`: the `default:` branch for an unrecognised `CDS_TEST_DB` prints "Unsupported CDS_TEST_DB value" but does not `os.Exit(1)`, so it falls through to `provider.SetTestDB(nil, dbType)` and a `teardown()` on a nil func.

---

## 19. Real-World Scenarios

**Current implementation note:** fuzzy matching and child-organization support described in §§12–13 and §18 are implemented behind configuration and organization policy, whereas commercial customer deployments, experiences and partner integrations below are use-case narratives and should not be read as verified production adoption.


> *Illustrative, drawn from requirements material and validated against current merge semantics.*

### Scenario A: Guest → Registered Shopper (Identity Stitching)

```
Before login (guest)
  [P_tmp_1] temporary
     ├─ profile_id = pid-abc
     ├─ user_id    = (none)
     ├─ traits: interest = shoes
     └─ application_data.web: cart_items = [A12, B44]

After login (stitched)
  [P_perm_U1] permanent / master
     ├─ user_id    = U1
     ├─ identity_attributes: email, mobile, name
     ├─ traits: interest = shoes
     ├─ application_data.web: cart_items = [A12, B44]
     └─ merged_from: [{ profile_id: pid-abc, reason: "system:user_id_match" }]
```

Schema needed:
- `application_data.web.cart_items` → `multi_valued: true`, `merge_strategy: combine`
- `identity_attributes.email` → `combine` (multi-valued) or `overwrite`
- `traits.interest` → `combine`

> `latest` is not a usable strategy — use `overwrite`.

### Scenario B: Multi-Channel Unification (Retail)

Four profiles:
- P1: web guest (`mobile=x`, `email=y`), `cart_items=[A12]`
- P2: web guest, later session (`mobile=x`), `cart_items=[B44]`
- P3: mobile app, logged in (`user_id=U1`, `email=y`), `wishlist=[W9]`
- P4: loyalty POS (`loyalty_id=L77`, `mobile=x`), `points=1200`

Rules by priority: userId (system) → mobile → email → loyalty ID.

Master (P3, the only permanent profile):
```
├─ user_id = U1
├─ identity_attributes: { email: y, mobile: x }
├─ traits: { interest: shoes }
├─ application_data:
│    ├─ web:     { cart_items: [A12, B44] }
│    ├─ mobile:  { wishlist: [W9] }
│    └─ loyalty: { loyalty_id: L77, points: 1200 }
└─ merged_from: [P1, P2, P4]
```

Because merging is pairwise and asynchronous, the profiles converge over several worker passes rather than in one shot — and only one rule fires per pass.

### Scenario C: Personalized Onboarding (SaaS signup funnel)

1. Visitor lands from a search for "AI agent access control" → `POST /profiles` creates a guest profile, cookie set
2. Context captured as traits (referrer, search keywords, device)
3. Browsing AI content updates `traits` with an interest tag
4. Newsletter signup writes `identity_attributes.email` onto the guest profile
5. Sign-up fires `POST_ADD_USER` with the cookie → `user_id` attached, prior preferences retained

Enables a personalized landing experience and filtered exports, e.g.
`?filter=identity_attributes.region eq China and traits.signupFor eq AI`

### Scenario D: Large Enterprise (Hotel Chain)

Three siloed systems: loyalty (keyed by personal email), booking (corporate email), on-site spa & dining (phone, unauthenticated).

CDS approach: **C1** as central master data authority ingesting from all three; **A2** unification rules merging across them without changing the legacy systems; **D** exposing the unified profile via consent-filtered APIs for personalized booking experiences.

---

## 20. MVP & Roadmap

**Reconciliation as of 2026-11-02:** the previously planned tolerant identity-resolution pipeline and B2B organization-management APIs **now exist in main** (see §§3–13 and 17–18). Treat prior roadmap references to *building* those foundations as historically motivated intent, not as a description of the current absence. Production adoption, advanced UI/explanations, confidence instrumentation, managed rollout and broader intelligence/IDV integrations remain separate initiatives and are not established by these commits.


### Implemented today

- **Profile management** — create/init anonymous profiles, cursor-paginated list with filters and attribute projection, get, patch, soft delete, `/Me` via cookie
- **Profile schema management** — per-scope attribute CRUD, typed validation, canonical values, complex sub-attributes
- **Unification rules** — CRUD with name and priority uniqueness
- **Profile unification** — async worker, `userId` invariant + priority-ordered rules, **bounded database-backed candidate discovery with deterministic oldest-match selection**, lazy application-data hydration for the selected pair, four merge cases, schema-driven attribute merging
- **Consent management** — categories, mandatory Identity Data, per-profile records, consent-filtered reads, system-app bypass
- **IS integration** — user lifecycle + session event sync, claim sync, event-driven and scheduled schema sync
- **Admin config** — per-org enablement and system-app registration
- **Ops** — health/readiness, process-owned bounded DB pools, request/worker cancellation propagation, graceful worker draining + DB-close ordering, cookie cleanup worker, pluggable deadline-aware queue shutdown, Helm chart with HPA/PDB, mTLS config
- **Zero-setup deployment** *(new in v0.3.63)* — inbuilt SQLite datasource, the default when `datasource.type` is unset. Schema embedded and applied on first start; no Docker, no external DB server. Dual-dialect query layer keeps the stores datasource-agnostic.

### Partially built / scaffolded

- Approval workflow for unification — `WAIT_ON_ADMIN` / `WAIT_ON_USER` statuses, `profile_unification_modes` and `profile_unification_triggers` tables, and `DefaultConfig()` exist, but nothing drives them and no API is exposed
- Merge strategies `ignore` / `latest` / `oldest` — partly implemented, not accepted by validation
- `computed` mutability — constant defined, not in the allowed set
- `origin_country` on `profiles` — column exists, no API path writes it

### Roadmap

- Probabilistic profile resolution (fuzzy matching)
- Synchronous profile resolution
- Profile unification approval workflow
- Data import and export
- Event ingestion to build profiles (profile enrichment from interactions)
- User segmentation (filter/export profiles by properties)
- Identity graph visualization
- MCP server for CDS
- Customer journey orchestration
- AI context support
- Verifiable Credential support

### Open items

- Authentication CDS → IS: system-app grant for org-specific tokens (`isSystemAppGrantEnabled` is present and defaults to `false`)
- Authorization is in-house — scope checking against JWT claims or introspection
- Guaranteed delivery on the message queue
- Console UI: editable profiles, account→profile linking, consent management screens
- End-to-end performance benchmarking (candidate-lookup microbenchmarks now exist up to 1M master profiles)
- Packaging: Go binary distribution alongside IS (cross-platform)
- Pricing model

---

*Reconciled against `wso2/identity-customer-data-service` `main` @ `a9fdadd78e5b575865ee9ed0d2ee99af40d6b3ae` (snapshot label `v0.4.1+a9fdadd`, regenerated 2026-10-26; previous baseline `49c5abc`). Sections 1, 2, 19 and the roadmap portion of 20 carry forward internal requirements material; everything else is derived from the source tree unless explicitly labelled otherwise. PR #330's testing claims are distinguished from directly verified code.*
