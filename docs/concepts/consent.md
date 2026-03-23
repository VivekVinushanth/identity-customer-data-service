# Consent

Consent controls which profile attributes a caller is allowed to **read**. When an app fetches a profile, it declares the consent category under which it is operating. The response is filtered to only the attributes the user has consented to under that category.

---

## Consent categories

A **consent category** is an org-level definition that declares:

- A human-readable name and a stable identifier
- The **purpose** of data use (`profiling`, `personalization`, or `destination`)
- The **attributes** this category covers — which fields of the profile an app operating under this consent is allowed to see

```json
{
  "category_name": "Marketing",
  "category_identifier": "marketing",
  "purpose": "personalization",
  "attributes": [
    { "scope": "identityAttributes", "attribute_id": "email" },
    { "scope": "identityAttributes", "attribute_id": "phone" },
    { "scope": "traits",             "attribute_id": "age" },
    { "scope": "applicationData",    "attribute_id": "last_purchase", "app_id": "com.acme.crm" }
  ]
}
```

### Attribute scopes

| Scope | Description |
|---|---|
| `identityAttributes` | Core identity fields (email, phone, name, etc.) |
| `traits` | Behavioural and preference fields |
| `applicationData` | Per-application data. Requires `app_id` to identify which app's data |

---

## Per-profile consent records

Each user profile has its own consent status per category. A consent record tracks:

- Which category the user has consented to
- Whether consent was given (`consent_status: true`) or revoked (`false`)
- When the consent was last recorded

Consent records are stored in `profile_consents` and managed via:

```
GET  /api/v1/{orgHandle}/profiles/{profileId}/consents
PUT  /api/v1/{orgHandle}/profiles/{profileId}/consents
```

---

## Consent-scoped profile fetch

When fetching a profile, callers pass one or more `consentId` query parameters. The response is filtered to only the attributes the user has actively consented to across the specified categories.

```
GET /api/v1/{orgHandle}/profiles/{profileId}?consentId=marketing
```

### Behaviour by case

| Scenario | Response |
|---|---|
| No `consentId` passed | Core fields only: `profile_id`, `user_id`, `meta` |
| `consentId` passed, user has consented | Attributes declared in that category's attribute list |
| `consentId` passed, user has not consented | That category contributes no attributes (silently) |
| Multiple `consentId`s passed | **Union** of allowed attributes across all passed categories the user has consented to |

### Multiple consent IDs — union

When multiple `consentId` values are passed, an attribute is returned if it appears in **any** of the specified categories the user has consented to. Categories the user has not consented to contribute nothing; the rest are unioned together.

```
GET /profiles/{id}?consentId=marketing&consentId=analytics

marketing covers:  email, phone, age
analytics covers:  email, age, last_login

union:             email, phone, age, last_login   ← all consented attrs returned
```

---

## Profile listing

The profile listing endpoint (`GET /profiles`) does **not** apply consent filtering. Listing is an administrative operation intended for console and system use, where the full profile is needed. Consent-scoped access applies only to single-profile fetch.

---

## What consent does NOT affect

| Area | Behaviour |
|---|---|
| Profile **writes** (update) | Consent does not gate write operations. Updates are controlled by permissions. |
| Profile **listing** | No consent filtering — full profiles returned. |
| Admin / system app access | Consent is enforced on API callers, not bypassed for system apps unless explicitly designed so. |

---

## Data model

```
consent_categories
  ├── category_identifier  (unique, stable ID)
  ├── purpose              (profiling | personalization | destination)
  └── consent_category_attributes
        ├── scope          (identityAttributes | traits | applicationData)
        ├── attribute_id   (references ProfileSchemaAttribute)
        └── app_id         (only for applicationData scope)

profile_consents
  ├── profile_id           → profiles
  ├── category_id          → consent_categories
  ├── consent_status       (true = consented, false = revoked)
  └── consented_at
```

---

## Sequence — consent-filtered profile fetch

```
Caller                    API                  ConsentFilterService         DB
  |                        |                          |                      |
  | GET /profiles/{id}     |                          |                      |
  |  ?consentId=marketing  |                          |                      |
  |----------------------->|                          |                      |
  |                        |-- fetch raw profile ----->|                      |
  |                        |                          |-------------------->|
  |                        |                          |<-- Profile ---------|
  |                        |                          |                      |
  |                        |-- filterByConsent() ----->|                      |
  |                        |   (profile, [marketing]) |                      |
  |                        |                          |-- profile_consents  |
  |                        |                          |   (has user consented?)
  |                        |                          |<-- consent rows ----|
  |                        |                          |                      |
  |                        |                          |-- category_attributes|
  |                        |                          |   (which attrs allowed?)
  |                        |                          |<-- attribute list ---|
  |                        |                          |                      |
  |                        |                          |-- filter profile    |
  |                        |<-- filtered profile ------|                      |
  |<-- 200 filtered -------|                          |                      |
```
