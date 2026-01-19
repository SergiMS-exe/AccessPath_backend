# AccessPath Backend API

Backend API for the AccessPath mobile application - helping people with disabilities find accessible places.

## Features

- User authentication (register/login with JWT)
- Search for accessible places by location, city, or category
- Add and manage places with accessibility information
- Rate and review places
- Comprehensive accessibility features tracking
- PostgreSQL database

## Tech Stack

- Node.js
- Express.js
- PostgreSQL
- JWT Authentication
- bcrypt for password hashing

## Getting Started

### Prerequisites

- Node.js (v14 or higher)
- PostgreSQL (v12 or higher)
- npm or yarn

### Installation

1. Clone the repository
2. Install dependencies:
   ```bash
   npm install
   ```

3. Create a `.env` file based on `.env.example`:
   ```bash
   cp .env.example .env
   ```

4. Update the `.env` file with your database credentials and configuration

5. Create the PostgreSQL database:
   ```bash
   createdb accesspath_db
   ```

6. Run the database migration:
   ```bash
   npm run migrate
   ```

7. Start the development server:
   ```bash
   npm run dev
   ```

The server will start on `http://localhost:3000`

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register a new user
- `POST /api/auth/login` - Login user
- `GET /api/auth/profile` - Get user profile (requires authentication)

### Places
- `GET /api/places` - Get all places (with filters: city, category, lat/lng/radius)
- `GET /api/places/:id` - Get place by ID
- `POST /api/places` - Create a new place (requires authentication)
- `PUT /api/places/:id` - Update place (requires authentication)
- `DELETE /api/places/:id` - Delete place (requires authentication)
- `POST /api/places/:id/features` - Add accessibility feature to place (requires authentication)
- `DELETE /api/places/:id/features/:featureId` - Remove feature from place (requires authentication)

### Reviews
- `GET /api/reviews/place/:placeId` - Get reviews for a place
- `POST /api/reviews` - Create a review (requires authentication)
- `PUT /api/reviews/:id` - Update review (requires authentication)
- `DELETE /api/reviews/:id` - Delete review (requires authentication)

### Accessibility Features
- `GET /api/features` - Get all accessibility features
- `GET /api/features/:id` - Get feature by ID
- `GET /api/features/category/:category` - Get features by category

## Database Schema

### Tables
- `users` - User accounts
- `places` - Accessible places
- `accessibility_features` - Types of accessibility features
- `place_accessibility_features` - Junction table linking places to features
- `reviews` - User reviews for places
- `photos` - Photos of places

## Scripts

- `npm start` - Start production server
- `npm run dev` - Start development server with nodemon
- `npm run migrate` - Run database migrations

## License

ISC
