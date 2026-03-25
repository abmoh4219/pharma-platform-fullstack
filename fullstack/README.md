# Pharmaceutical Compliance & Talent Operations Platform

Fully offline full-stack platform for recruitment, compliance, cases, and audit operations.

## Stack
- Frontend: Vue 3 + Vite + Tailwind CSS + shadcn-vue-style components
- Backend: Go + Gin
- Database: MySQL 8
- Orchestration: Docker Compose

## One-Click Startup
```bash
docker compose up
```

To rebuild images after code changes:
```bash
docker compose down && docker compose up --build
```

## Service Addresses
- Frontend: http://localhost:5173
- Backend: http://localhost:8080
- API base: http://localhost:8080/api/v1
- Health: http://localhost:8080/api/v1/health

## Test Login
- Username: `admin`
- Password: `Admin123!`

## Quick Verification
1. **Login**
   - Open frontend and sign in with admin credentials.
2. **Dashboard**
   - Confirm role/scope and summary counters load.
3. **Recruitment**
   - Create a position, create a candidate, run smart search, and test duplicate merge dialog.
4. **Compliance**
   - Create/update qualification and verify expiration countdown highlighting.
   - Create a restriction and run controlled medication rule check.
5. **Cases**
   - Create a case, inspect kanban status board, assign case, transition status, upload an attachment.
6. **Audit**
   - Search logs and export CSV.

## Tests
### Automatic on `docker compose up`
- `go-backend` entrypoint runs backend unit tests on startup.
- `test-runner` service runs `./run_tests.sh` once and prints a clear summary in container logs.
- Application services (`mysql`, `go-backend`, `vue-frontend`) keep running normally after tests finish.

### Manual
```bash
./run_tests.sh
```

## Security & Validation
- Error response format (standardized):
  - `{"success": false, "error": "message", "code": "ERROR_CODE"}`
- Validation strategy:
  - Gin binding + validation tags are enforced for body/query inputs.
  - Unknown JSON fields are rejected by the backend decoder.
  - Critical query/path parameters (IDs, status, pagination, date filters) are validated before execution.
- Security measures:
  - JWT authentication with strict token parsing and explicit expired/invalid handling.
  - Logout token invalidation with blacklist checks on protected routes.
  - RBAC + data-scope enforcement middleware across protected modules.
  - IP-based rate limiting middleware for API protection.
  - Security headers middleware (CSP, frame protection, XSS/content-type hardening, permission policy) + restricted CORS origins.
  - Consistent JSON errors for auth failures, unknown routes/methods, and panic recovery paths.
  - Gin trusted-proxy hardening (`SetTrustedProxies(nil)`) to prevent unsafe proxy trust defaults.
  - Request context logging with request ID, method, path, status, latency, and client IP.

## Project Notes
- Schema + seed data: `backend/migrations/init.sql`
- Seeded roles: business_specialist, compliance_admin, recruitment_specialist, system_admin
- Auth tokens expire in 8 hours and are invalidated on logout via token blacklist
- Sensitive fields are encrypted in DB and masked in responses

## Documentation
- Design overview: `docs/design.md`
- API endpoints: `docs/api-spec.md`
