# Healthy SaaS

A well-maintained SaaS application with TypeScript frontend and Python backend.

## Architecture

- **Frontend**: React + Next.js (TypeScript)
- **Backend**: FastAPI (Python)
- **Database**: PostgreSQL via SQLAlchemy

## Getting Started

```bash
# Frontend
cd frontend && npm install && npm run dev

# Backend
cd backend && pip install -r requirements.txt && uvicorn app.main:app
```

## Environment Variables

See `.env.example` for required configuration.

## Testing

```bash
# Frontend tests
cd frontend && npm test

# Backend tests
cd backend && pytest
```
