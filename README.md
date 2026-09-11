# Ticket System (Golang Backend Intern Assignment)

A small backend service where a user can register, log in, create tickets,
view only their own tickets, and update the status of their own tickets.

## Tech / Design Choices

- **Language:** Go (standard library only — no external Go modules). This
  was a deliberate choice to keep the project dependency-free, easy to
  build anywhere, and simple to review.
- **Storage:** In-memory, thread-safe store (`store.go`), guarded by a
  `sync.RWMutex`. Data resets when the process restarts. This is allowed
  by the assignment scope ("No complex database schema is required. Use
  in-memory storage, SQLite, PostgreSQL, or any simple persistent store").
- **Auth:** Hand-rolled HS256 JWT (`utils.go`) using only
  `crypto/hmac`, `crypto/sha256`, and `encoding/base64` — no external JWT
  library needed. Tokens are valid for 24 hours.
- **Password hashing:** Passwords are stored as `salt:sha256(salt+password)`
  (see `hashPassword` / `verifyPassword` in `utils.go`) — never in plain
  text. `bcrypt` was intentionally avoided since it needs an external
  package (`golang.org/x/crypto/bcrypt`); this keeps the project at zero
  external dependencies.
- **Routing:** Go 1.22's built-in `net/http.ServeMux`, which supports
  method + path-pattern routing (`GET /tickets/{id}`) natively, so no
  third-party router is needed either.

## Project Structure

```
.
├── main.go        # server setup, routing, port/secret config
├── models.go       # User, Ticket types, status flow rules
├── store.go        # in-memory thread-safe data store
├── auth.go         # /auth/register, /auth/login handlers
├── middleware.go    # JWT auth middleware
├── tickets.go       # ticket CRUD + status update handlers
├── utils.go         # JSON helpers, password hashing, JWT encode/decode
├── Dockerfile
├── .env.example
└── README.md
```

## API Endpoints

| Method | Endpoint               | Auth required | Purpose                    |
|--------|-------------------------|----------------|-----------------------------|
| GET    | `/health`               | No             | Health check                |
| POST   | `/auth/register`        | No             | Register a user             |
| POST   | `/auth/login`           | No             | Log in, get a JWT           |
| POST   | `/tickets`              | Yes            | Create a ticket             |
| GET    | `/tickets`              | Yes            | List your own tickets       |
| GET    | `/tickets/{id}`         | Yes            | Get one of your own tickets |
| PATCH  | `/tickets/{id}/status`  | Yes            | Update your own ticket's status |

Protected endpoints require an `Authorization: Bearer <token>` header.

### Request / response shapes

**POST /auth/register**
```json
// request
{ "username": "alice", "password": "secret123" }
// response (201)
{ "id": "usr_1", "username": "alice" }
```

**POST /auth/login**
```json
// request
{ "username": "alice", "password": "secret123" }
// response (200)
{ "token": "<jwt>" }
```

**POST /tickets** (requires `Authorization: Bearer <token>`)
```json
// request
{ "title": "Printer broken", "description": "Office printer jam" }
// response (201)
{
  "id": "tkt_1",
  "user_id": "usr_1",
  "title": "Printer broken",
  "description": "Office printer jam",
  "status": "open",
  "created_at": "2026-09-10T12:53:16Z",
  "updated_at": "2026-09-10T12:53:16Z"
}
```

**PATCH /tickets/{id}/status**
```json
// request
{ "status": "in_progress" }
```

Allowed status flow: `open -> in_progress -> closed`. A `closed` ticket can
never move back to `open` or `in_progress` (returns `409 Conflict` if you
try). Requests for a ticket you don't own return `404 Not Found` (not
`403`), so as not to leak whether the ticket exists at all.

## Local Run

Without Docker:
```bash
go build -o ticket-system .
JWT_SECRET=some-secret ./ticket-system
curl http://localhost:8080/health
```

With Docker (as required by the assignment):
```bash
docker build -t ticket-system .
docker run -p 8080:8080 ticket-system
curl http://localhost:8080/health
```

Expected health response:
```json
{ "status": "ok" }
```

## Environment Variables

See `.env.example`:

- `JWT_SECRET` — secret used to sign/verify JWTs. If not set, the app falls
  back to an insecure default and logs a warning — fine for local testing,
  **must** be set to a real secret in any real deployment.
- `PORT` — defaults to `8080`.

## Deployment

- **Deployed URL:** _fill in after deploying (e.g. Render / Railway / Fly.io free tier)_
- **Health check URL:** `<deployed-url>/health`

## Assumptions

- Users are identified by a unique `username` (not email) for register/login,
  since the assignment brief didn't specify exact field names for auth.
- IDs are simple incrementing strings (`usr_1`, `tkt_1`, ...) rather than
  UUIDs, to keep the in-memory store easy to read/debug.
- Ticket ownership check returns `404` (not `403`) for another user's
  ticket, to avoid confirming that a ticket ID exists at all.
- Minimum password length of 6 characters is enforced on registration as a
  basic sanity check; not explicitly required by the brief.
