# MultiBook Backend

Production-oriented Go backend for the MultiBook mobile application. Firebase remains responsible for authentication, storage and FCM; Supabase PostgreSQL is the system of record for application data.

## Local prerequisites

- Go 1.24 or newer
- Docker Desktop
- Firebase service-account credentials for local token verification

## Run locally

```bash
cp .env.example .env # Add local secrets only to .env; it is ignored by Git.
docker compose up -d postgres
go mod tidy
make migrate-up
go run ./cmd/api
```

The API listens on `http://localhost:8080`. Use `GET /healthz` for liveness and `GET /readyz` for database readiness.

## Commands

```bash
make test
make test-integration
make lint
make migrate-up
make migrate-supabase-up
make run
```

## Repository layout

```text
cmd/api/                   process entry point and graceful shutdown
internal/app/api/           HTTP router, global middleware and health endpoints
internal/platform/          external infrastructure: configuration, database and Firebase Auth
internal/shared/httpx/      shared HTTP response and error helpers only
internal/modules/           feature modules; each owns its domain and use cases
migrations/                ordered PostgreSQL migrations
api/openapi/               versioned API contract
deployments/               container and cloud deployment assets
```

The `users` module illustrates the standard module boundary:

```text
internal/modules/users/
  domain/                   domain types; no transport or database dependency
  application/              use cases plus command and query persistence ports
  adapters/http/            HTTP handlers and request-scoped current-user context
  adapters/postgres/        PostgreSQL command and query adapters
    commands.go             create, update, archive, ownership and other writes
    queries.go              list, detail, search and other read models
```

Dependencies point inward: adapters depend on `application` and `domain`; the application layer depends only on `domain` and its ports. Firebase token verification remains platform infrastructure in `internal/platform/auth`. Global HTTP concerns (request ID, recovery, logging, CORS, rate limiting, timeouts, routing) remain in `internal/app/api`.

Customer-facing business catalog reads belong to the query-only
`customer_discovery` module rather than the provider-owned `businesses`
aggregate. It owns `CustomerBusinessSummary` list projections,
`CustomerBusinessDetail`, discovery ranking, city and collection catalogs,
generic text search and stay recommendations. Its published
`CustomerBusinessSummaryReader` loads current active summaries by ID while
preserving caller order. The `saved_businesses` and `recently_viewed` modules
use this contract to hydrate customer-owned relations without duplicating
business snapshots.

## Command and query ports

Modules that both mutate and read persistent data define separate application ports: a named `XRepository` for commands and `XQueries` for read models. Services receive both dependencies; the router wires each to the matching PostgreSQL adapter. This keeps persistence writes such as create, replace, update, archive and ownership changes separate from list, detail and search queries.

Modules with only one responsibility expose only the relevant port. For example, `service_search` is query-only, while `business_locations` is command-only. HTTP handlers remain the sole boundary for HTTP and JSON mapping; application and PostgreSQL layers use typed domain or read models.

## Command transactions, audit and business lifecycle

Multi-step PostgreSQL commands use the typed `database.WithinTransaction` helper with a `pgx.Tx` callback. It is deliberately small: callback failure rolls the transaction back and successful work commits, without an ORM or a generic persistence framework. This is the standard command pattern for business aggregates, appointments, stay units, service staff/offering links and development seeds.

Business-scoped audit persistence is exposed through the typed `audit/application.Writer` port. Command adapters pass their active `pgx.Tx` to the port, so an audit event is committed only with the associated command and rolls back with it on failure. The PostgreSQL adapter persists to the existing `business_audit_events` table and supports optional structured metadata; it does not affect the external HTTP or JSON contract. It is used by businesses, locations, media, stays, stay unit types, service details, offerings, staff, availability and appointment commands, and is reusable by future booking and payment commands.

Business lifecycle rules are enforced in the application service before a command adapter starts changing nested data: ownership is checked first, a business type is immutable after creation, and an archived business cannot be edited.

`internal/shared` is deliberately small. A type belongs there only when it is a stable, cross-domain primitive used by multiple modules; domain models, repositories and services never go there.

## Naming conventions

| Layer | Convention | Example |
| --- | --- | --- |
| Go exported type and field | PascalCase | `UserRole`, `BusinessID`, `CreatedAt` |
| Go local variable | camelCase | `userRole`, `businessID` |
| Go package | lowercase | `users`, `bookings` |
| PostgreSQL table, column and enum type | snake_case | `business_locations`, `business_id`, `user_role` |
| JSON field | snake_case | `business_id`, `created_at` |

Go enum values use typed PascalCase constants, while their serialized PostgreSQL and JSON values stay lowercase snake_case where applicable.

## Media storage contract

PostgreSQL stores Firebase Storage paths and metadata, never Firebase download URLs. For example, a user avatar is stored as `profiles/<firebase_uid>/profile.webp` in `users.avatar_storage_path`.

The Go API returns that stable path. Flutter resolves it through the Firebase Storage SDK to a current download URL only when rendering an image. Download URLs are transient UI data: they are not persisted in PostgreSQL and are not sent back to the API.

## Provider business REST contract

Business data is owned by the authenticated provider and uses the existing Flutter `BusinessModel` camelCase map. No transport-only business DTOs are required on the client.

| Endpoint | Purpose | Payload/response |
| --- | --- | --- |
| `GET /v1/businesses` | Lightweight list for selectors, tabs and *My Businesses* | `{ items: BusinessSummaryResponse[] }` |
| `GET /v1/businesses/{businessID}` | Full aggregate, called only before opening the provider edit screen | Complete `BusinessModel` |
| `POST /v1/businesses` | Create a business | `BusinessModel.toMap()` → complete `BusinessModel` |
| `PUT /v1/businesses/{businessID}` | Replace a business aggregate | `BusinessModel.toMap()` → complete `BusinessModel` |

`PUT` reconciles nested stay/service data, including amenities, extras, rooms, offerings and staff. A client must first fetch `GET /v1/businesses/{businessID}` rather than editing the list summary; this keeps an unrelated edit from replacing omitted nested fields with empty collections.

The Flutter payload carries server UUIDs for existing rooms, offerings, providers and availability slots. `PUT` updates those rows in place, inserts entries with local draft IDs, and returns the resulting UUIDs. Omitted stay unit types, physical units, service staff and offerings are retired with `is_active=false` and `deleted_at` instead of being hard-deleted, preserving booking and appointment history. Amenities, extras, media, featured collections, staff-offering links and availability windows do not carry booking or appointment references and may be reconciled as value or join data.

## Customer discovery and development seed

`GET /v1/discovery/businesses` provides paginated customer-discovery cards for authenticated customers. It accepts `type` (`stays` or `services`), optional `city`, `limit` and `offset`, then returns lightweight camelCase `BusinessModel`-compatible cards sorted by rating and review count. Supplying `city` powers **Popular Near You**; omitting it returns the global ranking used by the default **Other stays** and **Other services** lists.

`GET /v1/discovery/cities` returns the alphabetized, de-duplicated permanent city catalog. A database trigger records a city whenever a business location is created or changed; cities are retained after businesses are archived or removed. The customer Explore city picker uses this route instead of Firestore.

`GET /v1/discovery/featured-collections?type=stays|services` returns the ordered Explore collection catalog from PostgreSQL. Each item includes its ID, image URL and localization keys; Flutter resolves those keys using its existing translations.

Recently Viewed is an independent customer-owned relation module.
`PUT /v1/recently-viewed/{businessID}` records an authenticated customer's
view of an active business. `GET /v1/recently-viewed?type=stays|services&limit=10`
returns that customer's latest active business summaries. PostgreSQL keeps only
the latest 30 views per customer, while `customer_discovery` provides their
current summary data.

### Customer Saved businesses

Saved is an independent, customer-owned relation module with authenticated,
customer-only endpoints:

| Method | Endpoint | Response |
| --- | --- | --- |
| PUT | `/v1/saved-businesses/{businessID}` | `204`; idempotently save an active business |
| DELETE | `/v1/saved-businesses/{businessID}` | `204`; idempotently remove the caller's reference, including unavailable businesses |
| GET | `/v1/saved-businesses/{businessID}` | `200 {"isSaved": true/false}`; unavailable/missing businesses return false |
| GET | `/v1/saved-businesses` | `200 {"items": [...]}`; all active saved stay/service cards, newest first |

Firebase token middleware authenticates requests; the application service requires a customer and adapters scope every query/mutation to the server-resolved user ID. Invalid UUIDs return `422`, non-customer roles `403`, and saving missing, inactive, archived or soft-deleted businesses `404`. No customer ID or business snapshot is accepted from the client.

Migration `000024_saved_businesses` stores only customer/business foreign keys and `saved_at`, with cascade cleanup, a composite primary key and an ordering index. Repeated PUT preserves the original ordering. Listing reuses the current business discovery read model and Flutter `BusinessModel` shape, hiding unavailable businesses. There is no Firestore fallback, stream, polling or WebSocket. Flutter emits a local refresh event only after successful REST mutations.

Apply with `make migrate-up` and `make migrate-supabase-up` after validation. Existing Firestore Saved snapshots are not imported by this schema migration; only REST saves populate the new table. Run `make test-integration` to verify database ownership isolation, idempotency, current business reads and lifecycle filtering.

### Business reviews

Reviews are owned by the independent `reviews` module and stored by migration
`000025_business_reviews`:

| Method | Endpoint | Response |
| --- | --- | --- |
| POST | `/v1/businesses/{businessID}/reviews` | `201`; creates a stay or service review from an eligible source |
| GET | `/v1/businesses/{businessID}/review-status` | `200 {"hasReview": true/false}` for the authenticated customer |
| GET | `/v1/businesses/{businessID}/reviews?limit=20&offset=0` | Newest-first review page with `items` and nullable `nextOffset` |

The API accepts only `sourceId`, `type` (`stay` or `service`), `rating` (1–5)
and an optional comment. Customer/business identity and display snapshots are
resolved server-side. A customer can review a business only once and only from
their own completed source, a stay whose checkout is before today, or a service
appointment whose end time has passed. Creation and the business rating/count
aggregate update run in one PostgreSQL transaction. Comments are trimmed,
empty comments become `null`, and content beyond 1000 Unicode characters is
truncated to preserve the previous callable behavior.

The Flutter client has no Firestore or callable fallback for reviews. Existing
Firestore review documents are not imported by the schema migration; migrate
historical data separately before production cutover if it must remain visible.

### Live dashboard and earnings metrics

Provider dashboard and earnings metrics belong to the independent, query-only
`metrics` module.

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | `/v1/businesses/{businessID}/dashboard-metrics` | Read the current dashboard snapshot |
| GET | `/v1/businesses/{businessID}/dashboard-metrics/stream` | Receive the initial and subsequent snapshots as Server-Sent Events |
| GET | `/v1/businesses/{businessID}/earnings?startDate=&endDate=&staffId=` | Read earnings for an inclusive creation-date range |
| GET | `/v1/businesses/{businessID}/earnings/stream?startDate=&endDate=&staffId=` | Stream the initial and subsequent earnings results |

The response uses explicit dashboard terminology: `activeReservationCount`,
`revenueMinor`, `reservationCount`, `dailyRevenueMinor` and
`dailyReservationCount`. Money remains in integer minor units and the response
includes the business currency. A reservation is active only while confirmed.
Current-month revenue and counts preserve the Firestore behavior: confirmed or
completed cash reservations contribute immediately, while non-cash
reservations contribute only after their payment status is paid. Values are
attributed to the reservation's UTC creation day.

Migration `000026_dashboard_metrics` adds only aggregation indexes and
transactional PostgreSQL notifications. It does not create a copied metrics
table or require initialization/backfill: booking and appointment tables stay
the source of truth. One API listener connection fans commit invalidations out
to SSE subscribers for the affected business, after which each subscriber
reloads a complete authoritative snapshot. The stream sends heartbeats during
idle periods and clients should reconnect normally if the connection closes.

Earnings uses the same committed-change invalidation stream and source tables.
It additionally returns online/cash totals and daily values. `staffId` is valid
only for a staff member of the owned service business; when supplied, the
response is filtered to that staff member and includes the historically stored
`provider_earnings_minor` value. The selected date range applies to reservation
`created_at`, preserving the business rule that revenue belongs to the moment
the booking or appointment was created. Calendar boundaries and daily buckets
use the client's validated `utcOffsetMinutes`, so local Monday/month/year
filters do not inherit the previous UTC date. Migration `000027_earnings_metrics`
adds the staff/date covering index needed by this query.


`GET /v1/discovery/search` powers customer text search. It accepts `type`, a required `query` and optional `limit`, then matches active businesses by normalized name or city prefix.

`GET /v1/discovery/recommended-stays` returns up to three personalized stay recommendations. It prioritizes the authenticated user's saved city, then fills any remaining positions with the global rating/review-count ranking so the carousel is populated even when the city has few or no stays.

Development environments expose provider-only `POST /v1/development/seed/stays` and `POST /v1/development/seed/services`. Seed media intentionally uses direct HTTPS image URLs rather than Firebase Storage paths, so dummy cards render without requiring Storage objects. The endpoints are disabled outside `APP_ENV=development`.

## Customer drafts REST contract

Customer booking and appointment drafts are stored in PostgreSQL and scoped to the authenticated customer. Migration `000017_customer_drafts` must be applied to the database used by the API before enabling these routes.

| Endpoint | Purpose | Response when no draft exists |
| --- | --- | --- |
| `GET /v1/drafts/booking` | Load the customer's single booking draft | `404` |
| `PUT /v1/drafts/booking` | Create or replace the booking draft | Current camelCase booking draft |
| `DELETE /v1/drafts/booking` | Remove the booking draft | `204` |
| `GET /v1/drafts/appointment` | Load the customer's single appointment draft | `404` |
| `PUT /v1/drafts/appointment` | Create or replace the appointment draft | Current camelCase appointment draft |
| `DELETE /v1/drafts/appointment` | Remove the appointment draft | `204` |

The routes require the `customer` role. A draft request carries only customer-selected values; the response enriches it with the active business name, location where applicable, price, provider name and business media path. `businessImageUrl` can therefore be a Firebase Storage path or a direct HTTPS URL. Client applications must resolve Storage paths only for display and must treat a `GET` `404` as an empty draft rather than a user-facing error.

## Customer stay bookings REST contract

Stay booking creation, customer history, cancellation and calendar availability are PostgreSQL-backed. Migrations `000015_stay_bookings` and `000018_stay_booking_extras` must be applied before these routes are enabled.

| Endpoint | Purpose | Response |
| --- | --- | --- |
| `POST /v1/businesses/{businessID}/stay/bookings` | Create a confirmed customer booking and calculate its server-authoritative totals | CamelCase `BookingModel`-compatible booking |
| `GET /v1/bookings?business_id=&cursor=&page_size=` | Page through the authenticated customer's stay bookings | `{ items: Booking[], nextCursor: string | null }` |
| `PATCH /v1/bookings/{bookingID}/cancel` | Cancel the customer's own confirmed booking through its checkout date | Updated camelCase booking |
| `GET /v1/businesses/{businessID}/stay/availability?room_type_id=` | Return date ranges that cannot be selected in the stay calendar | `{ unavailableRanges: [{ checkIn, checkOut }] }` |

All routes require the `customer` role. Booking history is customer-scoped even when `business_id` is supplied. Cancellation is rejected once checkout has passed, or when the booking is not the customer's confirmed booking. Availability never exposes another customer's booking data: for a single-unit stay it returns every confirmed occupied range; for a multiple-unit stay `room_type_id` is required and only fully occupied dates are returned.

The server owns prices, availability, payment status, confirmation codes and the selected-extras snapshot. The booking response is compatible with Flutter `BookingModel`, including `discountAmount`, `createdAt`, `updatedAt` and stored extra pricing data. Promotions are not yet a PostgreSQL backend module, so the current server calculation returns a zero discount.

## Customer service appointments REST contract

Service appointment creation, customer history, cancellation and rescheduling are PostgreSQL-backed.

| Endpoint | Purpose | Response |
| --- | --- | --- |
| `POST /v1/businesses/{businessID}/service/appointments` | Create a confirmed customer appointment and reserve its server-validated slot | CamelCase `AppointmentModel`-compatible appointment |
| `GET /v1/appointments?business_id=&status=&cursor=&page_size=` | Page through appointments visible to the authenticated actor | `{ items: Appointment[], nextCursor: string | null }` |
| `PATCH /v1/appointments/{appointmentID}/status` | Change an authorized appointment status, including customer cancellation | Updated camelCase appointment |
| `PATCH /v1/appointments/{appointmentID}/reschedule` | Move an authorized appointment to an available slot | Updated camelCase appointment |

For customer calls, the history is always scoped to the authenticated customer. Pagination uses the same offset cursor contract as customer stay bookings: the default page size is `20`, the maximum is `50`, and `nextCursor` is `null` at the end. List, status and reschedule responses contain the presentation fields and offerings required by Flutter `AppointmentModel`; they do not require client-side Firestore enrichment.

## Provider bookings REST contract

Providers use the same paginated list routes with `business_id`; ownership is enforced by the API. `GET /v1/bookings?business_id=&status=&cursor=&page_size=` returns only the provider's stay bookings, and `GET /v1/appointments?business_id=&status=&cursor=&page_size=` does the same for service appointments. `PATCH /v1/bookings/{bookingID}/status` permits provider `declined`, `completed` and eligible cash `no_show` transitions. Appointment status uses `PATCH /v1/appointments/{appointmentID}/status` with the equivalent provider transitions. Both routes return the updated Flutter-compatible model.

## Notifications REST contract

In-app notifications and FCM device registrations are PostgreSQL-backed. Apply migration `000019_notifications` before enabling these routes. Firebase remains the push transport: the API sends FCM messages to the registered device tokens after it persists a booking or appointment notification.

| Route | Purpose | Response |
| --- | --- | --- |
| `GET /v1/notifications?cursor=&page_size=` | Page through the authenticated user's in-app notifications | `{ items: Notification[], nextCursor: string \| null }` |
| `GET /v1/notifications/unread-count` | Read the authenticated user's unread count | `{ count: number }` |
| `PATCH /v1/notifications/{notificationID}/read` | Mark one of the user's notifications as read | `204` |
| `PUT /v1/notification-devices/{deviceID}` | Create or refresh an FCM device registration with `{ token }` | `204` |
| `DELETE /v1/notification-devices/{deviceID}` | Remove the current user's device registration | `204` |

Booking and appointment creation notify the owning provider. Provider status changes notify the customer. When a customer cancels, only the provider receives a dedicated cancellation event; the customer does not receive a notification about their own action. The API writes the in-app record first; push-delivery failure does not roll back the business action.

On Flutter, `NotificationDeviceService` registers the FCM token through the device routes. While the app is in the foreground, an FCM message makes `NotificationBellCubit` refresh `GET /v1/notifications/unread-count` immediately; the bell also uses a 30-second fallback refresh. This uses FCM plus a small REST read, not a WebSocket connection.

## Tests

`make test` runs unit and package tests. `make test-integration` starts the local PostgreSQL container, applies migrations and executes build-tagged integration tests. Those tests exercise database-enforced exclusion constraints, ownership queries and FK/reference safety: historical appointments prevent physical deletion of their staff and offerings, while physical stay units prevent deletion of their unit type. Supported archive operations use soft deletion or deactivation so those references remain valid.

## Security baseline

- Clients never connect to Supabase directly.
- Every protected route verifies a Firebase ID token server-side.
- Firebase service-account files and all `.env` files are ignored by Git.
- PostgreSQL credentials are supplied only via environment variables.
