# Assumptions and Design Decisions

This document records decisions made where the assignment does not provide an
explicit requirement. These assumptions keep the implementation consistent and
make the intended behavior clear to reviewers.

## General

1. **API format** - All application endpoints accept and return JSON.
2. **API versioning** - Business endpoints use the `/api/v1` prefix so future
   breaking changes can be introduced without changing existing clients.
3. **Date and time format** - Date-only fields, such as `date_of_birth`, use
   `YYYY-MM-DD` (for example, `1990-01-15`). Timestamps, such as `created_at`,
   use UTC with a timezone offset (for example, `2026-09-26T00:00:00Z`).
4. **Identifiers** - Internal entity IDs use PostgreSQL `BIGINT IDENTITY`
   columns. Hospital codes and patient identifiers remain human-readable
   external identifiers.
5. **Error format** - All API errors use a consistent object containing a
   machine-readable `code` and a human-readable `message`.
6. **Sensitive data** - Passwords, access tokens, national IDs, passport IDs,
   and complete patient records are not written to application logs.

## Database and Domain Models

1. **PostgreSQL schema** - The application uses one PostgreSQL schema,
   `public`, with three core models: `hospitals`, `staff`, and `patients`.
2. **Hospital ownership** - Every staff member and patient belongs to exactly
   one hospital through `hospital_id`.
3. **Hospital identity** - Each hospital has a unique, stable code such as
   `hospital-a`. API requests use this code instead of an internal numeric ID.
4. **Staff uniqueness** - A username is unique within a hospital, enforced by
   `UNIQUE (hospital_id, username)`. The same username may exist at different
   hospitals because login input includes the hospital.
5. **Patient uniqueness** - `patient_hn` is unique within a hospital, enforced
   by `UNIQUE (hospital_id, patient_hn)`, because different hospitals may issue
   the same HN value.
6. **Optional identifiers** - A patient may have a national ID, a passport ID,
   or both. Missing identifiers are stored as `NULL`, not empty strings.
7. **Patient persistence** - The `patients` table acts as a local cache of
   normalized records obtained from a hospital HIS. On an identifier cache miss,
   Agnos calls the hospital HIS and persists the response locally. The HIS
   integration remains behind a client interface so the persistence strategy can
   change without changing the HTTP layer.
8. **Initial hospital data** - Supported hospitals and their integration
   configuration are inserted through a database migration or seed process
   before staff accounts are created.

## Authentication and Authorization

1. **Password storage** - Staff passwords are hashed with bcrypt and are never
   stored or returned as plain text.
2. **Login result** - A successful login returns a signed JWT access token.
3. **Token claims** - The token contains the staff ID, hospital ID, and hospital
   code needed to authorize patient searches and select the hospital HIS client.
   It does not contain patient information.
4. **Token lifetime** - Access tokens expire after one hour. Refresh tokens are
   outside the assignment scope.
5. **Hospital scope** - Patient search derives `hospital_id` only from the
   authenticated staff token. A client cannot provide or override it in the
   request.
6. **Defense in depth** - The repository query also filters by `hospital_id`;
   authorization is not enforced only in the HTTP handler.
7. **Staff creation authorization** - `/staff/create` is unauthenticated for the
   assignment because no administrator role or provisioning flow is specified.
   In production, this endpoint would require an administrator or an invitation
   workflow.
8. **Login errors** - Invalid username, password, and hospital combinations
   return the same generic error so the API does not reveal which value exists.

## Patient Search

1. **HTTP method** - Patient search uses `POST /api/v1/patient/search`. Although
   the operation is read-only, POST keeps sensitive search values out of URLs,
   browser history, proxy logs, and common access logs.
2. **Search criteria** - Individual search fields are optional, but a request
   must contain at least one non-empty criterion. An empty search is rejected
   to prevent accidental retrieval of every patient in a hospital.
3. **Combined filters** - When multiple criteria are supplied, they are combined
   with `AND` so every provided criterion must match.
4. **Identifier matching** - National ID and passport ID use exact matching.
5. **Text matching** - Name fields use case-insensitive matching. Partial-name
   matching may be supported, but identifier fields never use partial matching.
6. **Pagination** - Search results are paginated with a default page size of 20
   and a maximum page size of 100.
7. **No results** - A valid search with no matching patients returns `200 OK`
   with an empty data array, not `404 Not Found`.
8. **Hospital isolation** - Search results include only patients whose
   `hospital_id` matches the authenticated staff member, regardless of the
   supplied criteria.

## Hospital Information System Integration

1. **Hospital-specific clients** - External HIS integrations implement a common
   interface. Hospital A is the first implementation.
2. **Hospital A identifiers** - Hospital A's
   `GET /patient/search/{id}` endpoint is called only when searching by national
   ID or passport ID, because those are the identifiers supported by the
   provided upstream API.
3. **Response normalization** - Hospital-specific response fields are converted
   into the application's shared `Patient` model before reaching handlers.
4. **Timeouts** - Outbound HIS requests time out after five seconds so an
   unavailable hospital system does not block application resources indefinitely.
5. **Upstream failures** - Timeout, connection, and invalid upstream-response
   errors are returned to the client as `502 Bad Gateway` using the standard
   error response. Internal upstream details are logged without patient data.
6. **Testing** - Unit tests replace the real HIS client with a mock or stub. Unit
   tests do not depend on Hospital A being reachable.

## Validation and API Behavior

1. **Required staff fields** - Staff creation and login require `username`,
   `password`, and `hospital`.
2. **Password policy** - A new password must contain at least eight characters.
3. **Gender values** - Patient gender follows the supplied HIS contract and is
   limited to `M` or `F` unless a future hospital integration requires a broader
   representation.
4. **Unknown fields** - Unknown JSON request fields are rejected to catch client
   mistakes early.
5. **Duplicate staff** - Creating an existing username in the same hospital
   returns `409 Conflict`.
6. **Unknown hospital** - Creating or logging in a staff member with an unknown
   hospital code returns an error without creating a new hospital implicitly.
