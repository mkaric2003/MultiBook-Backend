# Customer business detail API

`GET /v1/discovery/businesses/{businessID}` returns the complete active business aggregate for the customer stay and service detail screens.

The endpoint is authenticated, but is not owner-scoped: any signed-in user can read an active, non-archived business. It returns the Flutter `BusinessModel` response shape, including location, media, featured collections, stay amenities/extras/rooms or service offerings/providers/weekly availability.

Archived, inactive, missing, or malformed IDs return the standard API error response. Owner management continues to use `GET /v1/businesses/{businessID}`.
