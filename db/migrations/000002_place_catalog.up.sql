-- Catalogo externo de lugares (Overture Maps) + fallback comunitario, y
-- eliminacion de la integracion legacy con Google Places.
--
-- Objetivo: el buscador deja de llamar a Google. El catalogo se carga
-- mensualmente desde Overture y se consulta con PostgreSQL + pg_trgm. Los
-- sitios que no estan en el catalogo pueden crearlos los usuarios, pero nacen
-- con moderation_status = 'pending' hasta que se verifican.
--
-- Semantica (no confundir las dos):
--   moderation_status = 'verified'  -> el sitio es real (puede mostrarse)
--   published         = TRUE        -> ademas tiene datos de accesibilidad
--
-- El proyecto aun no esta en produccion, asi que la limpieza de Google se hace
-- en esta misma migracion en vez de en una incremental.
--
-- Ver docs/TECNICO_CATALOGO_LUGARES.md y docs/PLAN_OVERTURE_ASTURIAS.md.

SET search_path TO accesspath, public;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE SCHEMA IF NOT EXISTS geo;

-- =========================================================================
-- Catalogo externo (una fila por POI y proveedor)
-- =========================================================================
CREATE TABLE geo.place_catalog (
    id               TEXT NOT NULL,          -- GERS id (Overture) / fsq_place_id
    source           TEXT NOT NULL,          -- 'overture' | 'foursquare'
    name             TEXT NOT NULL,
    address          TEXT,
    city             TEXT,
    region           TEXT,
    postcode         TEXT,
    country          TEXT,
    latitude         DOUBLE PRECISION NOT NULL,
    longitude        DOUBLE PRECISION NOT NULL,
    category         TEXT,
    subcategory      TEXT,
    confidence       DOUBLE PRECISION,
    operating_status TEXT,
    phones           TEXT[],
    websites         TEXT[],
    refreshed_at     TIMESTAMPTZ,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source, id)
);

CREATE INDEX idx_catalog_name_trgm    ON geo.place_catalog USING gin (name gin_trgm_ops);
CREATE INDEX idx_catalog_address_trgm ON geo.place_catalog USING gin (address gin_trgm_ops);
CREATE INDEX idx_catalog_geo          ON geo.place_catalog (latitude, longitude);
CREATE INDEX idx_catalog_operating    ON geo.place_catalog (operating_status);

-- Historial de importaciones: permite auditar release, alcance y volumen.
CREATE TABLE geo.catalog_import (
    id          BIGSERIAL PRIMARY KEY,
    source      TEXT NOT NULL,
    release     TEXT NOT NULL,
    scope       TEXT NOT NULL,              -- 'asturias' | 'spain'
    row_count   INTEGER NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- place: origen del dato y estado de moderacion
-- =========================================================================
ALTER TABLE place ADD COLUMN source TEXT NOT NULL DEFAULT 'community';
ALTER TABLE place ADD COLUMN external_id TEXT;
ALTER TABLE place ADD COLUMN google_ftid TEXT;
ALTER TABLE place ADD COLUMN source_url TEXT;
ALTER TABLE place ADD COLUMN moderation_status TEXT NOT NULL DEFAULT 'verified';

ALTER TABLE place ADD CONSTRAINT place_moderation_status_check
    CHECK (moderation_status IN ('verified', 'pending', 'rejected'));

-- Deduplicacion por (proveedor, id externo). Parcial: los lugares creados por
-- usuarios no tienen external_id.
CREATE UNIQUE INDEX idx_place_source_external
    ON place (source, external_id)
    WHERE external_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_place_moderation ON place (moderation_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_place_name_trgm  ON place USING gin (name gin_trgm_ops);

-- =========================================================================
-- Limpieza de Google Places (legacy)
-- =========================================================================
-- Ya no hay ningun camino que llame a la API de Google: el buscador usa el
-- catalogo y los enlaces solo se parsean como pista. Sobran por tanto la
-- columna de deduplicacion antigua y el log de llamadas facturables.
DROP INDEX IF EXISTS idx_place_google_place_id;
ALTER TABLE place DROP COLUMN IF EXISTS google_place_id;
DROP TABLE IF EXISTS gmaps_api_log;
