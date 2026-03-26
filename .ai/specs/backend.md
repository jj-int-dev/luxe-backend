# Backend Spec

Architecture:

- Modular monolith
- Echo framework
- Clean architecture:
  handler → service → repository

Rules:

- No business logic in handlers
- Repository = DB only
- Services = logic only

Testing:

- TDD required
- Write tests first

Auth:

- Supabase JWT validation

Deployment:

- Render (handle cold starts)

Core Features:

- Items API
- Dream API
- Interactions
- Notifications
