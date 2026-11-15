-- Revierte 000002_place_catalog.
--
-- La extension pg_trgm NO se elimina: puede estar en uso por otras consultas
-- y su creacion es inocua. El catalogo geo si se elimina por completo.
--
-- Se recrean las estructuras de Google Places que esta migracion habia
-- borrado (columna, indice y tabla de log), para que el esquema vuelva a ser
-- equivalente al de 000001. Dos diferencias inevitables: pg_trgm se conserva
-- y google_place_id queda como ultima columna de place.

SET search_path TO accesspath, public;

DROP SCHEMA IF EXISTS geo CASCADE;

DROP INDEX IF EXISTS idx_place_name_trgm;
DROP INDEX IF EXISTS idx_place_moderation;
DROP INDEX IF EXISTS idx_place_source_external;

ALTER TABLE place DROP CONSTRAINT IF EXISTS place_moderation_status_check;
ALTER TABLE place DROP COLUMN IF EXISTS moderation_status;
ALTER TABLE place DROP COLUMN IF EXISTS source_url;
ALTER TABLE place DROP COLUMN IF EXISTS google_ftid;
ALTER TABLE place DROP COLUMN IF EXISTS external_id;
ALTER TABLE place DROP COLUMN IF EXISTS source;

-- Restaura el esquema legacy de Google Places de 000001.
ALTER TABLE place ADD COLUMN IF NOT EXISTS google_place_id TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_place_google_place_id
    ON place(google_place_id) WHERE google_place_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS gmaps_api_log (
    id        BIGSERIAL PRIMARY KEY,
    called_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gmaps_log_called_at ON gmaps_api_log(called_at);
