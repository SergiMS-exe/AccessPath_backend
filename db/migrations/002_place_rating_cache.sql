-- Migration: add place_rating_cache table and supporting indexes

CREATE TABLE IF NOT EXISTS place_rating_cache (
    place_id       BIGINT NOT NULL REFERENCES places(id) ON DELETE CASCADE,
    subcategory_id BIGINT NOT NULL REFERENCES subcategories(id) ON DELETE CASCADE,
    avg_score      NUMERIC(4,2) NOT NULL,
    total_ratings  INT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (place_id, subcategory_id)
);

CREATE INDEX IF NOT EXISTS idx_reviews_place_deleted
    ON reviews(place_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_review_ratings_review
    ON review_ratings(review_id);

CREATE INDEX IF NOT EXISTS idx_review_ratings_subcat
    ON review_ratings(subcategory_id);

CREATE INDEX IF NOT EXISTS idx_rating_cache_place
    ON place_rating_cache(place_id);
