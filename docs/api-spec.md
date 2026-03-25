# API Specification (v1)

Base URL: `/api/v1`

Response envelope:
- success: `{"success": true, "data": ...}`
- error: `{"success": false, "error": {"code": "...", "message": "..."}}`

## Public Endpoints
- `GET /health` — service and database health snapshot
- `POST /auth/login` — username/password login, returns JWT + user context

## Authenticated Endpoints

### Authentication
- `GET /auth/me` — current authenticated user
- `POST /auth/logout` — blacklist current token (`jti`)

### Dashboard
- `GET /dashboard/summary` — scoped operational counters

### Recruitment (`recruitment_specialist`, `system_admin`)
- `GET /recruitment/positions`
- `POST /recruitment/positions`
- `GET /recruitment/candidates`
- `POST /recruitment/candidates`
- `PUT /recruitment/candidates/:id`
- `POST /recruitment/candidates/import` (CSV/XLSX multipart)
- `POST /recruitment/candidates/merge`
- `GET /recruitment/candidates/search?q=...` (0-100 score + explanation)

### Compliance (`compliance_admin`, `system_admin`)
- `GET /compliance/qualifications`
- `POST /compliance/qualifications`
- `PUT /compliance/qualifications/:id`
- `DELETE /compliance/qualifications/:id`
- `GET /compliance/restrictions`
- `POST /compliance/restrictions`
- `PUT /compliance/restrictions/:id`
- `DELETE /compliance/restrictions/:id`
- `POST /compliance/restrictions/check`

### Cases (`business_specialist`, `compliance_admin`, `recruitment_specialist`, `system_admin`)
- `GET /cases`
- `POST /cases`
- `PUT /cases/:id/assign`
- `PUT /cases/:id/status`
- `GET /cases/:id/attachments`

### Files (chunked resumable upload)
- `POST /files/initiate`
- `POST /files/chunk`
- `POST /files/complete`
- `GET /files/sessions/:id`
- `GET /files/:id/download`

### Audit (`compliance_admin`, `system_admin`)
- `GET /audit/logs` (filter + pagination)
- `GET /audit/logs/export` (CSV)

## Notes
- All authenticated routes require `Authorization: Bearer <token>`.
- Data scope filtering (`institution/department/team`) is enforced by backend middleware.
- Audit logs are append-only and capture sensitive operations.
