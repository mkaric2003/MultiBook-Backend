# MultiBook Backend

## 1. Purpose and System Boundaries

MultiBook Backend is a production-oriented Go API for the MultiBook mobile application. It manages business data, authorization, reservations, availability, discovery, chat, notifications, and reporting projections.

The active system uses:

- Go 1.24 for the application process;
- Chi for HTTP routing and middleware;
- pgx for PostgreSQL access;
- PostgreSQL as the sole source of business data;
- Supabase as the hosted PostgreSQL environment;
- Firebase Admin SDK for ID token verification and FCM delivery;
- Firebase Storage as an external media service managed by the mobile client.

The mobile application accesses business data exclusively through this API. It does not connect directly to PostgreSQL or Supabase.

---

## 2. Architecture

```text
HTTP request
    │
    ▼
Global middleware
    │ request ID, recovery, logging, CORS, rate limit, timeout
    ▼
Firebase token authentication
    │ current user context
    ▼
Module HTTP adapter
    │ transport validation and mapping
    ▼
Application service
    │ authorization and business rules
    ▼
Command / query port
    │
    ▼
PostgreSQL adapter
```

Dependencies point inward:

- `domain` defines domain types and does not depend on HTTP or the database;
- `application` implements use cases and depends on the domain and ports;
- `adapters/http` maps HTTP/JSON to application calls;
- `adapters/postgres` implements command and query ports;
- `internal/platform` contains shared infrastructure such as configuration, database, and Firebase authentication;
- `internal/app/api` composes modules and the global HTTP pipeline.

There is no generic repository hiding different domains. Each module explicitly defines the ports required by its use cases.

### 2.1 Repository Structure

```text
cmd/api/                    process entrypoint and graceful shutdown
internal/app/api/           router, composition root, and middleware
internal/platform/          config, PostgreSQL, and Firebase infrastructure
internal/shared/httpx/      shared HTTP response and error helpers
internal/modules/           independent business modules
migrations/                 ordered PostgreSQL up/down migrations
api/openapi/                versioned public HTTP contract
deployments/                deployment configuration
Dockerfile                  multi-stage production image
docker-compose.yml          local PostgreSQL and migration tool
```

Standard module structure:

```text
internal/modules/<module>/
  domain/                   entities, value types, and domain constants
  application/              services and command/query ports
  adapters/http/            request/response mapping
  adapters/postgres/        SQL commands and queries
  adapters/background/      optional background processes
```

### 2.2 Command and Query Ports

A module that reads and modifies data separates responsibilities:

- `XRepository` covers create, update, archive, and other commands;
- `XQueries` covers list, detail, search, and read projections.

A query-only module exposes only a query port, while a command-only module exposes only a command port. The application service does not depend on pgx types or on the HTTP request.

---

## 3. HTTP Lifecycle and Security

All routes under `/v1` are protected. The request lifecycle is:

1. global middleware assigns a request ID and applies recovery, logging, CORS, rate limiting, and timeout;
2. the Firebase authenticator verifies the bearer ID token;
3. the verified `uid` becomes the request identity;
4. user middleware ensures an application profile exists;
5. the module verifies role, ownership, and domain rules;
6. only then is the query or command executed.

Security rules:

- a client-supplied user ID is never proof of identity;
- customer resources are always scoped to the authenticated customer;
- provider business operations require verified ownership;
- business type is immutable after creation;
- an archived business is not editable or visible in customer discovery;
- an availability response does not expose private data from other reservations;
- booking and appointment prices, discounts, and availability are calculated server-side;
- the review source must belong to the authenticated customer and business;
- the API does not accept a full card number or CVV;
- database credentials and Firebase service-account files come exclusively from a secure environment.

### 3.1 Standard HTTP Responses

Handlers use shared `httpx` helpers for JSON and errors. Typical statuses are:

| Status | Meaning |
|---|---|
| `200` | successful read or update |
| `201` | resource created |
| `204` | successful command without a response body |
| `400` | invalid request shape |
| `401` | missing or invalid ID token |
| `403` | role or ownership does not allow the operation |
| `404` | resource not found within the allowed scope |
| `409` | current-state conflict |
| `422` | domain or field validation |
| `429` | rate limit |
| `500` | unexpected server error |

`internal/app/api/router.go` is the source of truth for active routes. `api/openapi/openapi.yaml` contains the machine-readable part of the transport contract, but it does not yet cover all registered modules. Handler and characterization tests protect compatibility of the mobile JSON shape.

---

## 4. PostgreSQL and Data Integrity

PostgreSQL is the business source of truth. The schema is fully reproducible through ordered migrations.

Main data groups:

| Domain | Data |
|---|---|
| Identity | user profiles, role, and selected business |
| Businesses | core business, location, media, collections, and audit events |
| Stays | stay details, amenities, extras, unit types, and physical units |
| Services | service details, offerings, add-ons, staff, and weekly availability |
| Reservations | stay bookings, booking extras, and service appointments |
| Availability | manual staff blocks and time constraints |
| Customer state | drafts, saved, and recently viewed relationships |
| Commerce | promotions and payment-method metadata |
| Reviews | reviews and business rating aggregate |
| Communication | conversations, participant state, messages, and push outbox |
| Notifications | in-app items and device registrations |
| Support | support tickets |

### 4.1 Transactions

Multi-step commands use `database.WithinTransaction` with a typed `pgx.Tx` callback. An error rolls back the entire operation, while a successful callback is committed.

Transactional integrity includes, among other things:

- replacement of the business aggregate and nested data;
- appointment creation and time reservation;
- booking creation and snapshotting of extras data;
- promotion application and consumption;
- review creation and rating update;
- chat message sending, conversation preview, unread state, and push outbox;
- business audit event together with the corresponding command.

### 4.2 References and Lifecycle

- Entities referenced by historical reservations are not physically deleted; they are deactivated or soft-deleted.
- Foreign keys, unique constraints, and exclusion constraints protect ownership and time conflicts independently of the application process.
- Bookings and appointments store historical snapshots of prices, names, promotions, and commission.
- Money is stored as integer minor units.
- UTC timestamps are the persistent time basis; a client UTC offset is used only when a business report requires local calendar boundaries.

---

## 5. Business Modules and API

### 5.1 Users

`users` links a Firebase `uid` to the business profile. The profile is provisioned through an authenticated request, while the role remains application data.

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/v1/users/me` | current profile |
| `PATCH` | `/v1/users/me` | profile update |
| `PUT` | `/v1/users/me/role` | customer/provider selection |
| `PUT` | `/v1/users/me/selected-business` | active provider business |

### 5.2 Businesses

A provider business is an aggregate with location, media, stay or service details, and nested offerings.

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/v1/businesses` | owner-only lightweight summary |
| `GET` | `/v1/businesses/{businessID}` | full owner-only aggregate |
| `POST` | `/v1/businesses` | create |
| `PUT` | `/v1/businesses/{businessID}` | full replacement of the editable aggregate |
| `PATCH` | `/v1/businesses/{businessID}` | partial business update |
| `DELETE` | `/v1/businesses/{businessID}` | archive |

`PUT` reconciles rooms, offerings, staff, availability, amenities, extras, media, and collections. An existing UUID remains the identity of a nested row; a local draft ID marks a new item. A list summary is not allowed as the basis for aggregate replacement because it does not contain all nested data.

The `business_locations`, `business_media`, `stays`, `stay_units`, `services`, `service_offerings`, `service_staff`, and `service_availability` modules also expose granular owner-only endpoints under business scope.

### 5.3 Customer Discovery and Search

`customer_discovery` is the query-only boundary for the active customer catalog:

- `GET /v1/discovery/businesses`
- `GET /v1/discovery/businesses/{businessID}`
- `GET /v1/discovery/cities`
- `GET /v1/discovery/featured-collections`
- `GET /v1/discovery/recommended-stays`
- `GET /v1/discovery/search`
- `GET /v1/stays/search`
- `GET /v1/services/search`

The list returns a lightweight projection, while detail returns the full customer aggregate. Search and filters are executed server-side, including city, category, price, rating, date, time, amenities, collection, and availability where applicable.

The city catalog is normalized and deduplicated. Recommended stays prioritize the customer's city, while remaining slots are filled using global ranking. Featured collections return a stable ID, order, image, and localization keys.

### 5.4 Saved and Recently Viewed

`saved_businesses` stores the customer-business relationship. PUT and DELETE are idempotent, while the list hydrates the current discovery projection and hides inactive business items.

`recently_viewed` stores the customer's latest views, retains at most 30 relationships, and returns current business data when read.

### 5.5 Drafts

A customer has at most one booking draft and one appointment draft:

- `GET`, `PUT`, `DELETE /v1/drafts/booking`
- `GET`, `PUT`, `DELETE /v1/drafts/appointment`

A draft stores the customer's selection, while the read response adds the required business representation. A missing draft returns `404` and represents a normal empty state for the client.

### 5.6 Stay Bookings

| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/v1/businesses/{businessID}/stay/bookings` | server-authoritative create |
| `GET` | `/v1/bookings` | customer or provider list according to role and filters |
| `PATCH` | `/v1/bookings/{bookingID}/cancel` | customer cancellation |
| `PATCH` | `/v1/bookings/{bookingID}/status` | provider status transition |
| `GET` | `/v1/businesses/{businessID}/stay/availability` | unavailable date ranges |

Create locks the relevant inventory, checks overlap/capacity, calculates price and promotion, and stores the booking snapshot. A single-unit booking occupies the entire business, while multiple-unit availability depends on room-type quantity.

### 5.7 Service Appointments

| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/v1/businesses/{businessID}/service/appointments` | create and reserve slot |
| `GET` | `/v1/appointments` | customer or provider list |
| `PATCH` | `/v1/appointments/{appointmentID}/status` | role-aware status transition |
| `PATCH` | `/v1/appointments/{appointmentID}/reschedule` | authorized reschedule |
| `GET` | `/v1/businesses/{businessID}/service/staff/{staffID}/available-slots` | bookable start minute |

Available slots account for weekly availability, staff-offering compatibility, manual blocks, existing appointments, and the total duration of selected services. The database exclusion rule remains the final protection against concurrent overlap.

### 5.8 Promotions

Promotions are business-scoped resources under `/v1/businesses/{businessID}/promotions`. Supported types are percentage, fixed amount, and coupon code, together with period, minimum amount, optional minimum nights, usage limit, and active status.

Checkout uses `/active` for preview. Booking or appointment creation revalidates and applies the promotion inside the reservation transaction. The client total is never authoritative.

### 5.9 Reviews

- `POST /v1/businesses/{businessID}/reviews`
- `GET /v1/businesses/{businessID}/review-status`
- `GET /v1/businesses/{businessID}/reviews`

A customer can rate a business only from their own completed reservation. One review is allowed per customer and business. The rating aggregate is updated in the same transaction as review creation.

### 5.10 Payment Methods

`/v1/payment-methods` supports list, create, delete, and atomic setting of the default method. Only brand, last four digits, expiry, and cardholder name are stored. The API does not accept full PAN, CVV, or payment credentials. These data are not reusable payment tokens.

### 5.11 Support Tickets

`GET/POST /v1/support-tickets` are available to the customer. Create accepts category, subject, and message; identity, contact snapshot, and initial status are derived server-side.

---

## 6. Metrics and Server-Sent Events

`metrics` is a query-only module over the booking and appointment source tables. It does not maintain a copied metrics table.

| Endpoint | Purpose |
|---|---|
| `GET /v1/businesses/{businessID}/dashboard-metrics` | current dashboard snapshot |
| `GET /v1/businesses/{businessID}/dashboard-metrics/stream` | initial and subsequent dashboard snapshots |
| `GET /v1/businesses/{businessID}/earnings` | earnings for a range and optional staff |
| `GET /v1/businesses/{businessID}/earnings/stream` | live earnings snapshots |

A PostgreSQL commit notification only invalidates the business. The API then recalculates the authorized snapshot and sends it to the subscriber. One dedicated `LISTEN` connection distributes invalidations to active SSE clients, while heartbeat keeps an idle stream alive.

Dashboard uses the current month. Earnings accepts inclusive `startDate`, `endDate`, `utcOffsetMinutes`, and optional `staffId`. A cash confirmed/completed reservation is included immediately, an online reservation only when paid, while a no-show is excluded. The staff filter uses the historical `provider_earnings_minor` snapshot.

---

## 7. Chat

A conversation belongs to one customer and one specific business. The `chat` module stores immutable messages together with participant read, typing, and presence state.

| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/v1/conversations` | get-or-create conversation |
| `GET` | `/v1/conversations` | keyset-paginated list |
| `GET` | `/v1/conversations/unread-count` | participant unread aggregate |
| `GET` | `/v1/conversations/{conversationID}` | detail |
| `GET` | `/v1/conversations/{conversationID}/messages` | keyset-paginated messages |
| `POST` | `/v1/conversations/{conversationID}/messages` | idempotent send |
| `PATCH` | `/v1/conversations/{conversationID}/read` | read boundary |
| `PUT` | `/v1/conversations/{conversationID}/typing` | typing lease |
| `PUT` | `/v1/conversations/{conversationID}/presence` | active-viewer lease |
| `GET` | `/v1/chat/stream` | participant-scoped invalidations |

The client generates message UUIDs for idempotent retries. Sending a message, updating the conversation preview, updating recipient unread state, and creating any push outbox record are committed together.

SSE carries only `chat_sync` and `chat_changed`; message content remains behind the authorized REST read endpoint. Reconnect begins with a sync event.

An active recipient in the same conversation does not receive an unread increment or push job. The typing lease lasts four seconds, while the presence lease lasts 45 seconds.

### 7.1 Chat Push Outbox

A background worker processes `chat_push_outbox`:

- a job is claimed with `FOR UPDATE SKIP LOCKED`;
- multiple API instances can safely share the queue;
- an unavailable device marks the job as skipped;
- a successful FCM submission marks it as sent;
- a transport error uses exponential retry from five seconds up to 15 minutes;
- there are at most eight attempts;
- a processing claim older than five minutes can be reclaimed after a process crash.

---

## 8. Notifications

In-app notifications and FCM device registrations belong to the `notifications` module.

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/v1/notifications` | paginated user list |
| `GET` | `/v1/notifications/unread-count` | unread count |
| `PATCH` | `/v1/notifications/{notificationID}/read` | mark as read |
| `PUT` | `/v1/notification-devices/{deviceID}` | upsert FCM token |
| `DELETE` | `/v1/notification-devices/{deviceID}` | remove device |

Booking and appointment events persist the in-app record before the push attempt. A push error does not roll back the business action. An idempotent notification ID prevents duplicates during retry.

A new reservation notifies the business provider. A provider status change notifies the customer. Customer cancellation notifies the provider, but the customer does not receive a notification for their own action. Chat uses a separate push outbox and does not create an in-app duplicate.

---

## 9. Media Storage Contract

PostgreSQL stores a Firebase Storage path, never a download URL. Examples:

```text
profiles/<firebase_uid>/profile.webp
businesses/<owner_uid>/<business_uuid>/<file>.webp
```

The Go API receives and returns the stable path. Flutter resolves it into the current download URL only for rendering. This contract prevents persistent coupling of data to a revocable media token and keeps binary content transfer outside the Go process.

A direct HTTPS URL is supported only where it is part of controlled development seed content.

---

## 10. Configuration

Configuration comes from environment variables:

| Variable | Purpose | Default |
|---|---|---|
| `APP_ENV` | environment and development-only capabilities | `development` |
| `PORT` | HTTP port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | required |
| `DB_MAX_CONNS` | maximum pool size | `10` |
| `DB_MIN_CONNS` | minimum pool size | `2` |
| `FIREBASE_PROJECT_ID` | Firebase Admin project | required |
| `GOOGLE_APPLICATION_CREDENTIALS` | local service-account path | environment-specific |
| `CORS_ALLOWED_ORIGINS` | comma-separated allowlist | empty if not set |
| `RATE_LIMIT_RPS` | requests per second | `20` |
| `RATE_LIMIT_BURST` | allowed burst | `40` |

`.env` and service-account files are not committed to Git. Cloud runtime should use a secret manager or platform workload identity capability where available.

---

## 11. Local Development

Prerequisites:

- Go 1.24 or newer;
- Docker Desktop;
- a Firebase service account for local token verification and FCM.

Run:

```bash
cp .env.example .env
docker compose up -d postgres
make migrate-up
make run
```

The API listens on `http://localhost:8080` according to the default configuration.

| Endpoint | Purpose |
|---|---|
| `GET /healthz` | process liveness |
| `GET /readyz` | PostgreSQL readiness |

### 11.1 Migrations

```bash
make migrate-up
make migrate-down
make migrate-supabase-up
make migrate-supabase-down
make migrate-supabase-force VERSION=<version>
```

Each schema change gets a new numbered `.up.sql` and `.down.sql` file. An applied migration is not modified retroactively. `force` is used only after determining the actual state of the schema version table.

---

## 12. Build and Deployment

Local build:

```bash
go build ./cmd/api
```

`Dockerfile` uses a Go Alpine build stage and a minimal distroless non-root runtime. The production image:

- does not contain the Go toolchain;
- runs `/api` as a non-root user;
- receives configuration exclusively through the environment;
- exposes port 8080;
- expects external PostgreSQL and Firebase credential setup.

Schema migrations are executed as a controlled deployment step, separately from API process startup.

---

## 13. Testing and Observability

```bash
make test
make test-integration
make lint
make tidy
```

- `make test` runs package and unit tests.
- `make test-integration` starts local PostgreSQL, applies migrations, and runs the integration suite.
- `make lint` uses `go vet`.
- `make tidy` aligns module dependencies.

Integration tests cover ownership isolation, database constraints, idempotency, concurrency-sensitive availability, lifecycle filtering, and reference safety.

The API uses structured `slog`. The global request logger includes request ID, method, path, status, and duration. Recovery middleware converts a panic into a controlled response and logs diagnostic context. `/healthz` and `/readyz` separate process state from database readiness.

---

## 14. Performance and Reliability

- The connection pool has configurable minimum and maximum limits.
- List endpoints use offset or keyset pagination according to the domain pattern.
- Discovery uses lightweight read projections instead of full aggregates.
- Search, availability, and metrics queries rely on dedicated indexes.
- Business aggregate commands use one transaction.
- PostgreSQL constraints remain the final protection under concurrent requests.
- SSE invalidations do not carry large business payloads.
- A dedicated LISTEN connection fans out events without one database connection per subscriber.
- Heartbeat keeps idle SSE connections alive, while the client performs reconnects.
- The transactional chat outbox separates reliable push retry from request latency.
- FCM failure does not invalidate an already successful business command.
- The rate limiter and request timeout limit uncontrolled resource consumption.

---

## 15. Development Rules

- One module owns its domain and persistence ports.
- HTTP request/response types remain in the HTTP adapter.
- The application service does not know about `http.Request`, the JSON decoder, or the pgx pool.
- The PostgreSQL adapter does not produce user-facing HTTP messages.
- Command and query implementations remain in separate files when a module has both responsibilities.
- A multi-step mutation must be transactional.
- A new business mutation must explicitly verify role and ownership.
- Money is transferred and stored in minor units.
- Dates and times have explicit UTC or local semantics.
- Errors are wrapped with context; the original cause is preserved.
- A shared package accepts only stable cross-domain primitives, not domain models.
- A new endpoint must be added to the router, OpenAPI, and corresponding handler/application tests.
- A new schema change must have both an up and a down migration.

---

## 16. Current Limitations

- Payment-method data are not payment tokens. Production charging requires a PCI-compliant provider, tokenization, webhook handling, and an idempotent payment lifecycle.
- Advanced full-text search may require a PostgreSQL FTS/trigram strategy or a separate index as volume and ranking requirements increase.
- OpenAPI currently does not fully cover all routes registered in the router and should be completed before relying on automatic client generation.
- The production environment should define metrics, tracing, alerting, backup/restore verification, and log retention policy.
- Firebase service account, APNs/FCM, and database credentials must be delivered through platform secret infrastructure.

---

## 17. Reference Files

- [`cmd/api/main.go`](cmd/api/main.go) — process lifecycle.
- [`internal/app/api/router.go`](internal/app/api/router.go) — composition root and HTTP routes.
- [`internal/app/api/middleware.go`](internal/app/api/middleware.go) — global middleware.
- [`internal/platform/config/config.go`](internal/platform/config/config.go) — environment configuration.
- [`internal/platform/database`](internal/platform/database) — pool and transaction helper.
- [`internal/platform/auth/firebase.go`](internal/platform/auth/firebase.go) — token verification and Messaging client.
- [`internal/modules`](internal/modules) — business modules.
- [`api/openapi/openapi.yaml`](api/openapi/openapi.yaml) — HTTP contract.
- [`migrations`](migrations) — PostgreSQL schema history.
- [`Makefile`](Makefile) — local and CI commands.
- [`Dockerfile`](Dockerfile) — production image.
