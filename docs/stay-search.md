# Stay search API

`GET /v1/stays/search` is the authenticated server-side filter endpoint used by the Flutter stay filter sheet and its quick chips.

Only explicitly selected filters need to be supplied. `page_size` and `offset` are pagination controls; all other parameters are optional.

| Parameter | Meaning |
| --- | --- |
| `city` | Normalized exact city match. |
| `check_in`, `check_out` | ISO dates. Availability is evaluated only when both dates are present. |
| `adults`, `children` | Required capacity when date availability is evaluated. |
| `min_price_minor`, `max_price_minor` | Nightly-price range in minor currency units. |
| `minimum_rating` | Inclusive minimum average rating. |
| `category_id`, `collection_id`, `amenity` | Repeatable values; amenities are combined with AND semantics. |
| `inventory_type` | `singleUnit` or `multipleUnits`. |
| `offset`, `page_size` | Offset pagination; page size is 1–20 and defaults to 8. |

The response has a Flutter `BusinessModel`-compatible `items` array and a `nextCursor` offset string or `null`. A property-type filter does not require a location; location is only used when `city` is supplied.
