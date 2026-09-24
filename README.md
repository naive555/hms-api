# Hospital Middleware System (HMS)

A middleware API that lets hospital staff search patient records from their own hospital only.
Patients are served from PostgreSQL. When a national ID or passport ID isn't found locally,
the API fetches the patient from the hospital's HIS, stores it, and returns it.

**Stack:** Go 1.27 · Gin · PostgreSQL 16 · Nginx · Docker Compose

**Design document** (project structure, API spec, ER diagram, design decisions):
https://docs.google.com/document/d/1MKKSOuHK9MWwKbMlpZ8PXtDvwiHf0dcr1d1nLx7JDaQ/edit?usp=sharing

---

## Quick start

Requirements: Docker with Compose v2. Go 1.27 only if you want to run the tests on the host.

```bash
git clone https://github.com/naive555/hms-api.git
cd hms-api
cp .env.example .env
docker compose up --build
```

The API is available at **http://localhost** (through nginx). Compose starts:

| Service   | Role                                                               |
|-----------|--------------------------------------------------------------------|
| `nginx`   | Only public entry point (`:80`). Login rate limit, JSON error pages |
| `app`     | Go API (`:8080`, not exposed to the host)                           |
| `db`      | PostgreSQL 16                                                       |
| `migrate` | Applies schema + demo seed data, then exits                         |
| `mockhis` | Stand-in for Hospital A's HIS (the real URL isn't reachable)        |

## Try it

The examples use [`jq`](https://jqlang.org) to pick fields out of the JSON.

```bash
# 1. Create one staff member in each hospital
curl -s -X POST localhost/staff/create -H 'Content-Type: application/json' \
  -d '{"username":"nurse_a","password":"secret123","hospital":"hospital-a"}'
curl -s -X POST localhost/staff/create -H 'Content-Type: application/json' \
  -d '{"username":"nurse_b","password":"secret123","hospital":"hospital-b"}'

# 2. Log in
TOKEN_A=$(curl -s -X POST localhost/staff/login -H 'Content-Type: application/json' \
  -d '{"username":"nurse_a","password":"secret123","hospital":"hospital-a"}' | jq -r .access_token)
TOKEN_B=$(curl -s -X POST localhost/staff/login -H 'Content-Type: application/json' \
  -d '{"username":"nurse_b","password":"secret123","hospital":"hospital-b"}' | jq -r .access_token)

# 3. Same search, different hospitals: each staff member only sees their own patients
curl -s "localhost/patient/search?first_name=som" -H "Authorization: Bearer $TOKEN_A" | jq '.total, [.data[].last_name_en]'
# 3  ["Jaidee", "Rakdee", "Wongsawat"]
curl -s "localhost/patient/search?first_name=som" -H "Authorization: Bearer $TOKEN_B" | jq '.total, [.data[].last_name_en]'
# 2  ["Srisuk", "Jaidee"]

# 4. HIS fallback: this national ID exists only in the mock HIS
curl -s "localhost/patient/search?national_id=1100500077777" -H "Authorization: Bearer $TOKEN_A" | jq '.data[0].first_name_en'
# "Wichai"   (now stored in hospital A's database)

# 5. No token
curl -s localhost/patient/search
# {"error":{"code":"UNAUTHORIZED","message":"missing or invalid token"}}
```

### Demo data

| Hospital     | Patients                                                                           |
|--------------|------------------------------------------------------------------------------------|
| `hospital-a` | Somchai Jaidee, Somying Rakdee, John Michael Smith (passport only), Somsak Wongsawat |
| `hospital-b` | Somchai Srisuk, Somchai Jaidee (same national ID as in A), Yuki Tanaka (passport only) |

The mock HIS (hospital A) knows two patients that are **not** seeded, to demo the fallback:
national ID `1100500077777` and passport `CD9876543`. Special IDs demo failure handling:

| `national_id`   | Mock HIS behaviour | API result                         |
|-----------------|--------------------|------------------------------------|
| `9999999999999` | 404                | `200`, empty list                  |
| `0000000000500` | 500                | `200`, empty list (error is logged) |
| `0000000000504` | responds after 10s | `200`, empty list after the 3s timeout |

Hospital B's HIS URL intentionally points to a host that doesn't exist, so ID searches there
show the same graceful degradation.

## API

| Method | Path              | Auth   | Input                                                                                         |
|--------|-------------------|--------|-----------------------------------------------------------------------------------------------|
| POST   | `/staff/create`   | –      | JSON `{username, password, hospital}`                                                         |
| POST   | `/staff/login`    | –      | JSON `{username, password, hospital}` → `{access_token, token_type, expires_in}`              |
| GET    | `/patient/search` | Bearer | Query (all optional): `national_id, passport_id, first_name, middle_name, last_name, date_of_birth, phone_number, email, limit, offset` |
| GET    | `/healthz`        | –      | –                                                                                             |

Every error uses the same body: `{"error": {"code": "...", "message": "..."}}`.
The design document has full request and response examples, validation rules and error codes.

## Tests

```bash
make test     # all unit tests
make cover    # coverage of internal/ (excludes cmd/)
```

Every layer has unit tests, with positive and negative cases for every endpoint:

- **handler:** `httptest` through the real router and auth middleware, with fake services
- **service:** in-memory fake repositories and HIS
- **repository:** `pgxmock`, which asserts the exact SQL and arguments
- **HIS client:** `httptest` server covering 200, 404, 500, timeout and malformed payloads
- **auth / middleware:** expired, forged, `alg:none` and wrong-algorithm tokens

Coverage is **97.7%** of `internal/`. `cmd/` is excluded: it holds only wiring (`main.go`)
and the mock HIS.

## Security and isolation

- The hospital scope comes **only from the JWT**. `/patient/search` has no hospital parameter,
  and every patient query starts with `WHERE hospital_id = $1`.
- Patients fetched from the HIS are always saved under the searching staff member's hospital.
- Passwords are hashed with bcrypt. An unknown username and a wrong password return the same
  error and take the same time, so usernames can't be enumerated.
- JWTs accept HS256 only and must carry `exp`.
- Search input is always passed as query arguments, never concatenated into SQL.
  `%` and `_` in names are matched literally.
- nginx limits login to 10 requests/minute per IP (burst 5), caps request bodies at 16 KB,
  and hides its version.

## Local development

```bash
make up          # start only Postgres
make migrate-up  # apply migrations (needs the golang-migrate CLI)
make run         # run the API on the host with values from .env
```

When running on the host, hospital A's HIS URL (`http://mockhis:8081`) only resolves inside
the Compose network. ID searches still work but fall back to local data.

### Configuration

| Variable       | Default | Description                                     |
|----------------|---------|-------------------------------------------------|
| `APP_PORT`     | `8080`  | API listen port                                 |
| `DATABASE_URL` | –       | PostgreSQL connection string (required)         |
| `JWT_SECRET`   | –       | HMAC secret, at least 32 characters (required)  |
| `JWT_TTL`      | `1h`    | Token lifetime (Go duration)                    |
| `HIS_TIMEOUT`  | `3s`    | Timeout for HIS calls                           |

## Notes

- **pgx is pinned to v5.10.0:** `pgxmock` v4.9.0 doesn't implement `pgx.Rows.TypeMap()` yet,
  which pgx v5.11 added. Upgrade both together once pgxmock catches up.
- `/staff/create` is public because the assignment asks for it. In production it should be
  limited to admins.
