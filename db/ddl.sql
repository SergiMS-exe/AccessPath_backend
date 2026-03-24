-- DDL (Data Definition Language) - Schema definitions

-- Users
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    code          UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    username      VARCHAR(100) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

-- Categories (accesibilidad: Física, Sensorial, Psíquica…)
CREATE TABLE IF NOT EXISTS categories (
    id            BIGSERIAL PRIMARY KEY,
    code          UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    name          VARCHAR(100) NOT NULL,
    slug          VARCHAR(100) NOT NULL UNIQUE,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Subcategories (características concretas de cada categoría)
CREATE TABLE IF NOT EXISTS subcategories (
    id            BIGSERIAL PRIMARY KEY,
    code          UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    category_id   BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    slug          VARCHAR(100) NOT NULL UNIQUE,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Places
CREATE TABLE IF NOT EXISTS places (
    id          BIGSERIAL PRIMARY KEY,
    code        UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    name        VARCHAR(255) NOT NULL,
    address     TEXT,
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    description TEXT,
    created_by  BIGINT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- Collections (listas de lugares guardados por el usuario)
CREATE TABLE IF NOT EXISTS collections (
    id         BIGSERIAL PRIMARY KEY,
    code       UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Collection places (pivot)
CREATE TABLE IF NOT EXISTS collection_places (
    collection_id BIGINT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    place_id      BIGINT NOT NULL REFERENCES places(id) ON DELETE CASCADE,
    added_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (collection_id, place_id)
);

-- Reviews
CREATE TABLE IF NOT EXISTS reviews (
    id         BIGSERIAL PRIMARY KEY,
    code       UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    place_id   BIGINT NOT NULL REFERENCES places(id) ON DELETE CASCADE,
    comment    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Review photos
CREATE TABLE IF NOT EXISTS review_photos (
    id         BIGSERIAL PRIMARY KEY,
    code       UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    review_id  BIGINT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Review ratings (una puntuación por subcategoría y reseña)
CREATE TABLE IF NOT EXISTS review_ratings (
    review_id      BIGINT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    subcategory_id BIGINT NOT NULL REFERENCES subcategories(id) ON DELETE CASCADE,
    score          INT NOT NULL CHECK (score >= 1 AND score <= 5),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (review_id, subcategory_id)
);

CREATE TABLE IF NOT EXISTS place_rating_cache (
    place_id       BIGINT NOT NULL REFERENCES places(id) ON DELETE CASCADE,
    subcategory_id BIGINT NOT NULL REFERENCES subcategories(id) ON DELETE CASCADE,
    avg_score      NUMERIC(4,2) NOT NULL,
    total_ratings  INT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (place_id, subcategory_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_places_created_by     ON places(created_by);
CREATE INDEX IF NOT EXISTS idx_places_location        ON places(latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_places_deleted_at      ON places(deleted_at);
CREATE INDEX IF NOT EXISTS idx_collections_user_id    ON collections(user_id);
CREATE INDEX IF NOT EXISTS idx_reviews_place_id       ON reviews(place_id);
CREATE INDEX IF NOT EXISTS idx_reviews_user_id        ON reviews(user_id);
CREATE INDEX IF NOT EXISTS idx_reviews_deleted_at     ON reviews(deleted_at);
CREATE INDEX IF NOT EXISTS idx_review_photos_review   ON review_photos(review_id);
CREATE INDEX IF NOT EXISTS idx_review_ratings_review  ON review_ratings(review_id);
CREATE INDEX IF NOT EXISTS idx_subcategories_category ON subcategories(category_id);
CREATE INDEX IF NOT EXISTS idx_reviews_place_deleted  ON reviews(place_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_review_ratings_review  ON review_ratings(review_id);
CREATE INDEX IF NOT EXISTS idx_review_ratings_subcat  ON review_ratings(subcategory_id);
CREATE INDEX IF NOT EXISTS idx_rating_cache_place     ON place_rating_cache(place_id);
