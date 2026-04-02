# Resume Optimizer Backend

A Go backend API that analyzes resumes against job descriptions using OpenAI, providing match scores, keyword analysis, skill extraction, section feedback, ATS format checks, and rewrite suggestions.

## Tech Stack

- **Language**: Go 1.25
- **Framework**: Gin
- **Database**: PostgreSQL (GORM)
- **Auth**: Supabase JWT
- **AI**: OpenAI API (GPT)

## Project Structure

```
├── main.go              # Entry point, router setup
├── config/              # Database connection
├── domain/              # Entities (structs) and repository interfaces
├── repository/          # Database access layer
├── usecase/             # Business logic layer
├── handler/             # HTTP request/response layer
├── service/             # External service integrations (OpenAI)
├── middleware/           # Auth middleware
└── doc/                 # OpenAPI specification
```

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL
- Supabase project (for JWT auth)
- OpenAI API key (optional, falls back to template-based suggestions)

### Setup

1. Clone the repository
2. Copy `.env` and configure environment variables:
   ```
   DATABASE_URL=postgres://user:pass@localhost:5432/dbname
   SUPABASE_JWT_SECRET=your-jwt-secret
   OPENAI_API_KEY=your-openai-key
   OPENAI_MODEL=gpt-4o-mini
   CORS_ORIGINS=http://localhost:3000
   PORT=8080
   ```
3. Run the server:
   ```bash
   go run main.go
   ```

The server starts on `http://localhost:8080`. Database tables are auto-migrated on startup.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/hello` | Health check |
| POST | `/analyses` | Create a new resume analysis |
| GET | `/analyses` | List analyses (paginated, sortable, filterable) |
| GET | `/analyses/{id}` | Get analysis by ID |
| DELETE | `/analyses/{id}` | Delete an analysis |
| POST | `/analyses/{id}/versions` | Create a new analysis version |
| PATCH | `/analyses/{id}/status` | Update application status |
| GET | `/profile` | Get user profile |
| PUT | `/profile` | Update/create profile |

All endpoints except `/hello` require `Authorization: Bearer <token>`.

See [doc/openapi.yaml](doc/openapi.yaml) for the full OpenAPI specification.
