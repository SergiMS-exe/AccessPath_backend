-- Database schema for AccessPath

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Places table
CREATE TABLE IF NOT EXISTS places (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    address VARCHAR(500) NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    country VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20),
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    phone VARCHAR(50),
    website VARCHAR(500),
    category VARCHAR(100),
    overall_accessibility_rating DECIMAL(3, 2) DEFAULT 0,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Accessibility features table
CREATE TABLE IF NOT EXISTS accessibility_features (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    icon VARCHAR(100),
    category VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Place accessibility features junction table
CREATE TABLE IF NOT EXISTS place_accessibility_features (
    id SERIAL PRIMARY KEY,
    place_id INTEGER REFERENCES places(id) ON DELETE CASCADE,
    feature_id INTEGER REFERENCES accessibility_features(id) ON DELETE CASCADE,
    is_available BOOLEAN DEFAULT true,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(place_id, feature_id)
);

-- Reviews table
CREATE TABLE IF NOT EXISTS reviews (
    id SERIAL PRIMARY KEY,
    place_id INTEGER REFERENCES places(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5) NOT NULL,
    comment TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Photos table
CREATE TABLE IF NOT EXISTS photos (
    id SERIAL PRIMARY KEY,
    place_id INTEGER REFERENCES places(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    url VARCHAR(500) NOT NULL,
    caption TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_places_location ON places(latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_places_city ON places(city);
CREATE INDEX IF NOT EXISTS idx_places_category ON places(category);
CREATE INDEX IF NOT EXISTS idx_reviews_place_id ON reviews(place_id);
CREATE INDEX IF NOT EXISTS idx_photos_place_id ON photos(place_id);
CREATE INDEX IF NOT EXISTS idx_place_accessibility_features_place_id ON place_accessibility_features(place_id);

-- Insert default accessibility features
INSERT INTO accessibility_features (name, description, category, icon) VALUES
('Wheelchair Ramp', 'Ramp access for wheelchairs', 'mobility', 'ramp'),
('Elevator', 'Elevator available', 'mobility', 'elevator'),
('Accessible Parking', 'Designated accessible parking spaces', 'mobility', 'parking'),
('Accessible Restroom', 'Restroom designed for accessibility', 'mobility', 'restroom'),
('Automatic Doors', 'Doors that open automatically', 'mobility', 'door'),
('Wide Doorways', 'Doorways wide enough for wheelchairs', 'mobility', 'doorway'),
('Braille Signage', 'Signs with Braille text', 'visual', 'braille'),
('Audio Assistance', 'Audio guidance or assistance available', 'visual', 'audio'),
('Service Animal Friendly', 'Service animals are welcome', 'general', 'pet'),
('Hearing Loop', 'Induction loop system for hearing aids', 'hearing', 'hearing'),
('Sign Language', 'Staff trained in sign language', 'hearing', 'sign-language'),
('Quiet Space', 'Designated quiet areas available', 'sensory', 'quiet'),
('Low Stimulation', 'Low sensory stimulation environment', 'sensory', 'low-light'),
('Accessible Seating', 'Seating designed for accessibility', 'mobility', 'chair'),
('Step-Free Access', 'Entrance without steps', 'mobility', 'no-stairs')
ON CONFLICT (name) DO NOTHING;
