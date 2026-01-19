# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AccessPath is a TypeScript-based REST API backend for a mobile application that helps people with disabilities find accessible places. The API provides user authentication, place management with geospatial search, accessibility feature tracking, and review systems.

## Development Commands

### Setup
```bash
npm install                    # Install dependencies
cp .env.example .env          # Create environment file
createdb accesspath_db        # Create PostgreSQL database
npm run migrate               # Run database migrations
```

### Running the Application
```bash
npm run dev                   # Start development server with hot reload
npm run build                 # Compile TypeScript to JavaScript
npm start                     # Start production server (requires build first)
```

### Database
```bash
npm run migrate               # Run migrations (creates tables and seeds default accessibility features)
```

**Important**: The migration script is idempotent - it uses `CREATE TABLE IF NOT EXISTS` and `INSERT ... ON CONFLICT DO NOTHING`, so it's safe to run multiple times.

## Architecture

### TypeScript Structure

The codebase uses **strict TypeScript** with comprehensive type definitions located in `src/types/`:

- **user.types.ts**: User entities, auth inputs/outputs
- **place.types.ts**: Place entities, search filters, geospatial queries
- **review.types.ts**: Review entities with user relationships
- **feature.types.ts**: Accessibility feature definitions
- **auth.types.ts**: JWT payload structures
- **express.types.ts**: Extended Express Request type with authenticated user

**Critical**: All Express request handlers that require authentication must use `AuthenticatedRequest` type (not standard `Request`) to access the `req.user` property safely.

### Layer Architecture

```
Routes → Controllers → Models → Database
  ↓          ↓           ↓
Validation  Business   Data Access
            Logic      + Types
```

**Routes** (`src/routes/`): Define endpoints with express-validator validation chains. Apply `authMiddleware` for protected routes and `validate` middleware to check validation results.

**Controllers** (`src/controllers/`): Handle HTTP request/response logic. All methods are `static async` and return `Promise<void>`. Must explicitly call `res.json()` or `res.status().json()` before returning (TypeScript enforces this).

**Models** (`src/models/`): Encapsulate database queries with typed results. All methods are `static async` and use typed `pool.query<Type>()` calls. Return types are strictly defined from `src/types/`.

**Middleware** (`src/middleware/`):
- `auth.ts`: JWT verification, attaches `user` to request
- `errorHandler.ts`: Global error handler with PostgreSQL error code handling
- `validation.ts`: Express-validator result checker

### Database Design

**Key relationships**:
- `places` → `users` (created_by, optional reference)
- `places` ↔ `accessibility_features` (many-to-many via `place_accessibility_features`)
- `reviews` → `places` (CASCADE delete)
- `reviews` → `users` (SET NULL delete)

**Geospatial queries**: The `places.findAll()` method includes a **Haversine formula** for radius-based searches using latitude/longitude coordinates. Query parameters: `lat`, `lng`, `radius` (in kilometers).

**Pre-seeded data**: Migration automatically inserts 15 accessibility features across 5 categories (mobility, visual, hearing, sensory, general).

### Authentication Flow

1. User registers/logs in via `/api/auth/register` or `/api/auth/login`
2. JWT token generated with `{ userId: number }` payload
3. Client includes token in `Authorization: Bearer <token>` header
4. `authMiddleware` verifies token and attaches full user object to `req.user`
5. Protected routes check `req.user.id` for authorization

**Security notes**:
- Passwords hashed with bcrypt (10 rounds)
- JWT secret from `JWT_SECRET` environment variable
- Token expiration from `JWT_EXPIRES_IN` (default: 7d)

## Environment Configuration

Required `.env` variables:
```
PORT=3000
NODE_ENV=development
DB_HOST=localhost
DB_PORT=5432
DB_NAME=accesspath_db
DB_USER=<your_user>
DB_PASSWORD=<your_password>
JWT_SECRET=<secure_random_string>
JWT_EXPIRES_IN=7d
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:19006
```

## Common Patterns

### Adding a New API Endpoint

1. Define types in `src/types/<domain>.types.ts`
2. Add model method in `src/models/<domain>.model.ts` with typed query
3. Add controller method in `src/controllers/<domain>.controller.ts`
4. Add route in `src/routes/<domain>.routes.ts` with validation
5. Register route in `src/app.ts`

### Database Queries

Always use parameterized queries with typed results:
```typescript
const result = await pool.query<UserType>(
  'SELECT * FROM users WHERE id = $1',
  [userId]
);
return result.rows[0];
```

### Error Handling

Controllers use try-catch with `next(error)`. The global error handler in `errorHandler.ts` automatically handles:
- PostgreSQL constraint violations (23505, 23503)
- Validation errors
- Custom errors with `statusCode` property

## API Authentication

**Public endpoints**: `/api/features/*`, `GET /api/places/*`, `GET /api/reviews/place/:placeId`

**Protected endpoints**: All `POST`, `PUT`, `DELETE` operations require `Authorization: Bearer <token>` header.

**Authorization checks**: Reviews and user-created content enforce ownership - users can only modify their own reviews.

## Database Schema Notes

- All tables use `SERIAL` primary keys (auto-incrementing integers)
- Timestamps use `TIMESTAMP` with `DEFAULT CURRENT_TIMESTAMP`
- Cascading deletes: reviews/photos cascade when place deleted
- Soft references: created_by uses `ON DELETE SET NULL` to preserve content
- Indexes on: location columns, city, category, foreign keys
