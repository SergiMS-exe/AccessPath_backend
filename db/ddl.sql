-- DDL (Data Definition Language) - Schema definitions

-- Users
CREATE TABLE IF NOT EXISTS "user" (
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
CREATE TABLE IF NOT EXISTS category (
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
CREATE TABLE IF NOT EXISTS subcategory (
    id            BIGSERIAL PRIMARY KEY,
    code          UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    category_id   BIGINT NOT NULL REFERENCES category(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    slug          VARCHAR(100) NOT NULL UNIQUE,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Places
-- Migration for existing DBs:
--   ALTER TABLE place ADD COLUMN IF NOT EXISTS google_place_id TEXT UNIQUE;
--   ALTER TABLE place ADD COLUMN IF NOT EXISTS published BOOLEAN NOT NULL DEFAULT FALSE;
--   UPDATE place SET published = TRUE WHERE EXISTS (SELECT 1 FROM review r WHERE r.place_id = place.id AND r.deleted_at IS NULL);
-- published: un lugar solo aparece en el mapa una vez tiene su primera valoracion.
-- Hasta entonces vive en la tabla (sirve de cache anti-duplicados de Google) pero oculto.
CREATE TABLE IF NOT EXISTS place (
    id              BIGSERIAL PRIMARY KEY,
    code            UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    name            VARCHAR(255) NOT NULL,
    address         TEXT,
    latitude        DOUBLE PRECISION NOT NULL,
    longitude       DOUBLE PRECISION NOT NULL,
    description     TEXT,
    google_place_id TEXT UNIQUE,
    published       BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      BIGINT NOT NULL REFERENCES "user"(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- Collections (listas de lugares guardados por el usuario)
CREATE TABLE IF NOT EXISTS collection (
    id         BIGSERIAL PRIMARY KEY,
    code       UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Collection places (pivot)
CREATE TABLE IF NOT EXISTS collection_place (
    collection_id BIGINT NOT NULL REFERENCES collection(id) ON DELETE CASCADE,
    place_id      BIGINT NOT NULL REFERENCES place(id) ON DELETE CASCADE,
    added_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (collection_id, place_id)
);

-- Reviews
CREATE TABLE IF NOT EXISTS review (
    id         BIGSERIAL PRIMARY KEY,
    code       UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    place_id   BIGINT NOT NULL REFERENCES place(id) ON DELETE CASCADE,
    comment    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Review photos
CREATE TABLE IF NOT EXISTS review_photo (
    id         BIGSERIAL PRIMARY KEY,
    code       UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    review_id  BIGINT NOT NULL REFERENCES review(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Review ratings (una puntuación por subcategoría y reseña)
CREATE TABLE IF NOT EXISTS review_rating (
    review_id      BIGINT NOT NULL REFERENCES review(id) ON DELETE CASCADE,
    subcategory_id BIGINT NOT NULL REFERENCES subcategory(id) ON DELETE CASCADE,
    score          INT NOT NULL CHECK (score >= 1 AND score <= 5),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (review_id, subcategory_id)
);

CREATE TABLE IF NOT EXISTS place_rating_cache (
    place_id       BIGINT NOT NULL REFERENCES place(id) ON DELETE CASCADE,
    subcategory_id BIGINT NOT NULL REFERENCES subcategory(id) ON DELETE CASCADE,
    avg_score      NUMERIC(4,2) NOT NULL,
    total_ratings  INT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (place_id, subcategory_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_place_created_by          ON place(created_by);
CREATE INDEX IF NOT EXISTS idx_place_location            ON place(latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_place_deleted_at          ON place(deleted_at);
CREATE INDEX IF NOT EXISTS idx_place_google_place_id     ON place(google_place_id) WHERE google_place_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_place_published_location   ON place(latitude, longitude) WHERE published AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_collection_user_id        ON collection(user_id);
CREATE INDEX IF NOT EXISTS idx_review_place_id           ON review(place_id);
CREATE INDEX IF NOT EXISTS idx_review_user_id            ON review(user_id);
CREATE INDEX IF NOT EXISTS idx_review_deleted_at         ON review(deleted_at);
CREATE INDEX IF NOT EXISTS idx_review_photo_review       ON review_photo(review_id);
CREATE INDEX IF NOT EXISTS idx_review_rating_review      ON review_rating(review_id);
CREATE INDEX IF NOT EXISTS idx_review_rating_subcat      ON review_rating(subcategory_id);
CREATE INDEX IF NOT EXISTS idx_subcategory_category      ON subcategory(category_id);
CREATE INDEX IF NOT EXISTS idx_review_place_deleted      ON review(place_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rating_cache_place        ON place_rating_cache(place_id);

-- Google Maps API call log (quota tracking)
-- Migration: CREATE TABLE IF NOT EXISTS gmaps_api_log (id BIGSERIAL PRIMARY KEY, called_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS gmaps_api_log (
    id         BIGSERIAL PRIMARY KEY,
    called_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gmaps_log_called_at ON gmaps_api_log(called_at);
