# API Documentation

This folder contains the Swagger/OpenAPI definition for the backend API.

## Files
- `openapi.yaml`: OpenAPI 3.0 specification for implemented endpoints

## Endpoints Included
- `GET /hello`
- `POST /analyses`
- `GET /analyses`
- `GET /analyses/{id}`
- `DELETE /analyses/{id}`
- `GET /profile`
- `PUT /profile`

## How to View
You can open `openapi.yaml` in:
- Swagger Editor: https://editor.swagger.io/
- VS Code OpenAPI extensions

## Auth
Protected endpoints use Bearer JWT auth (`Authorization: Bearer <token>`).
