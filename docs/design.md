# Platform Design Overview

## Purpose
The Pharmaceutical Compliance & Talent Operations Platform is a fully offline enterprise system that unifies:
- identity and access control (JWT + RBAC + data scope),
- recruitment workflows,
- compliance workflows,
- case ledger operations,
- append-only audit tracking.

## Architecture
- **Frontend:** Vue 3 + Vite + Tailwind CSS + shadcn-vue-style components
- **Backend:** Go + Gin REST API
- **Database:** MySQL 8
- **Containers:** Docker Compose (`mysql`, `go-backend`, `vue-frontend`, `test-runner`)

## Security Model
- Login uses username/password with bcrypt hash verification.
- Access token is JWT (8-hour expiry).
- Logout writes JWT `jti` into token blacklist; revoked tokens are blocked.
- RBAC enforced per route for:
  - `business_specialist`
  - `compliance_admin`
  - `recruitment_specialist`
  - `system_admin`
- Data-scope constraints enforced by `institution / department / team`.
- Sensitive fields are encrypted at rest and masked in API responses.

## Core Modules
- **Dashboard:** scope-aware operational metrics.
- **Recruitment:** positions, candidates, bulk import, duplicate merge, smart scoring search.
- **Compliance:** qualification lifecycle, expiration countdown, controlled medication rules.
- **Cases:** unique case numbering, duplicate submission block, assignment/status transitions, attachments.
- **Audit:** searchable append-only records and CSV export.

## Frontend UX Design
- Modern blue/green pharma palette.
- Responsive app shell with role-based sidebar and contextual header.
- Card-driven layouts, hover feedback, loading indicators, confirmation dialogs, and toast notifications.
- Kanban-style case status board plus operational tables/forms.

## Testing Strategy
- Frontend component unit tests (Vitest).
- Backend unit tests (Go test).
- API smoke scripts for login, recruitment search, and case creation.
- Unified test launcher: `run_tests.sh`.
- Tests run once automatically at startup via Docker Compose `test-runner` service.
