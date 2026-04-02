# API Documentation

This folder contains the Swagger/OpenAPI definition for the backend API.

## Files

- `openapi.yaml`: OpenAPI 3.0 specification for implemented endpoints

## Endpoints Included

### Health
- `GET /hello` - Health check

### Analyses (auth required)
- `POST /analyses` - Create a new resume analysis
- `GET /analyses` - List analyses (paginated, sortable, filterable via `?page=&limit=&sort=&order=&company=&position=&status=`)
- `GET /analyses/{id}` - Get one analysis by ID (all versions)
- `DELETE /analyses/{id}` - Delete an analysis
- `POST /analyses/{id}/versions` - Create a new version for an existing analysis
- `PATCH /analyses/{id}/status` - Update application status

### Profile (auth required)
- `GET /profile` - Get current user profile
- `PUT /profile` - Update or create profile (upsert)

## How to View

You can open `openapi.yaml` in:

- Swagger Editor: https://editor.swagger.io/
- VS Code OpenAPI extensions

## Auth

Protected endpoints use Bearer JWT auth (`Authorization: Bearer <token>`).
