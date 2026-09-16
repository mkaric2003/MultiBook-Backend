# Service search API

`GET /v1/services/search` is the authenticated REST replacement for the Firebase `searchServices` callable.

All filters are optional. The endpoint returns a Flutter `BusinessModel`-compatible `items` array and an opaque `nextCursor` string (or `null`). Results are evaluated for availability before they are sorted and paginated, matching the previous Firebase behavior.

| Parameter | Meaning |
| --- | --- |
| `appointment_date` | ISO date (`YYYY-MM-DD`). Requires `start_minutes`. |
| `start_minutes` | Appointment start, in 30-minute minutes-from-midnight increments. Without a date it is ignored, as it was in Firebase. |
| `category_id`, `collection_id`, `city` | Optional exact filters. City matching is trim/case/diacritic insensitive. |
| `min_price_minor`, `max_price_minor` | Inclusive offering price range in minor currency units. |
| `sort_option` | `recommended`, `priceLowToHigh`, `priceHighToLow`, or `rating`. |
| `cursor`, `page_size` | Cursor pagination. Page size is 1–20 and defaults to 8. |

When date and time are set, a business is returned only if at least one active staff member can perform one full active offering in a weekly availability window that does not overlap a confirmed appointment or an availability block.
