-- DDL (Data Definition Language) - Schema completo AccessPath v2
--
-- Modelo de contribuciones atomicas sobre dimensiones funcionales.
-- Reescrito desde cero (clean break, pre-lanzamiento): no hay migracion
-- incremental de las tablas del modelo v1 (category/subcategory/review_rating/
-- place_rating_cache/review). Para regenerar la BD basta con ejecutar este
-- script y luego dml.sql sobre una base vacia.
--
-- Todas las tablas viven en el esquema accesspath. El search_path se fija a
-- nivel de BD para que el codigo y el seed usen nombres sin cualificar.

CREATE SCHEMA IF NOT EXISTS accesspath;
DO $$ BEGIN
    EXECUTE format('ALTER DATABASE %I SET search_path TO accesspath, public', current_database());
END $$;
SET search_path TO accesspath, public;

-- =========================================================================
-- Usuarios
-- =========================================================================
CREATE TABLE "user" (
    id                                BIGSERIAL PRIMARY KEY,
    code                              UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    username                          VARCHAR(100) NOT NULL,
    email                             VARCHAR(255) NOT NULL UNIQUE,
    password_hash                     VARCHAR(255) NOT NULL,
    -- Consentimiento explicito para el perfil funcional (necesidades). NULL = sin consentimiento.
    accessibility_profile_consent_at  TIMESTAMPTZ,
    created_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                        TIMESTAMPTZ
);

-- Necesidades funcionales del usuario (opt-in, dato sensible, NO diagnostico).
-- need_key debe casar con criterion.profile_tags.
CREATE TABLE user_profile_need (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    need_key   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, need_key)
);

-- =========================================================================
-- Lugares
-- =========================================================================
-- published: un lugar nace oculto (sirve de cache anti-duplicados de Google por
-- google_place_id) y pasa a visible con su primera contribucion.
CREATE TABLE place (
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

-- =========================================================================
-- Catalogo del formulario (dimensiones -> criterios -> opciones)
-- =========================================================================
CREATE TABLE dimension (
    id          BIGSERIAL PRIMARY KEY,
    code        UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    key         TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT,
    sort_order  INT NOT NULL DEFAULT 0,
    active      BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE criterion (
    id                      BIGSERIAL PRIMARY KEY,
    code                    UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    dimension_id            BIGINT NOT NULL REFERENCES dimension(id) ON DELETE CASCADE,
    key                     TEXT NOT NULL UNIQUE,      -- 'aseos.existe'
    prompt                  TEXT NOT NULL,             -- texto UI (con acentos ok)
    profile_tags            TEXT[],                    -- {'silla','baja_vision',...}
    weight                  SMALLINT NOT NULL DEFAULT 1,
    is_blocking             BOOLEAN NOT NULL DEFAULT FALSE,
    is_starter              BOOLEAN NOT NULL DEFAULT FALSE,
    -- Solo se pregunta si el criterio del que depende salio exists=true.
    depends_on_criterion_id BIGINT REFERENCES criterion(id),
    sort_order              INT NOT NULL DEFAULT 0,
    active                  BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE answer_option (
    id            BIGSERIAL PRIMARY KEY,
    criterion_id  BIGINT NOT NULL REFERENCES criterion(id) ON DELETE CASCADE,
    label         TEXT NOT NULL,          -- 'Hay rampa pero es dificil'
    exists_value  BOOLEAN,                -- true/false; NULL = "no se"
    quality_value SMALLINT CHECK (quality_value BETWEEN 1 AND 5),
    is_unsure     BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order    INT NOT NULL DEFAULT 0
);

-- =========================================================================
-- Capa cualitativa: submission (envoltorio opcional de comentario + fotos)
-- =========================================================================
-- submission: la valoracion VIVA de UN usuario sobre UN lugar. Unica por
-- (user, place); nace al pulsar "Quiero valorar" y se edita a lo largo del
-- tiempo. Comentario y fotos son campos opcionales; las contribuciones cuelgan
-- de ella (contribution.submission_id NOT NULL).
CREATE TABLE submission (
    id         BIGSERIAL PRIMARY KEY,
    code       UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    place_id   BIGINT NOT NULL REFERENCES place(id) ON DELETE CASCADE,
    comment    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Una valoracion viva por usuario y lugar.
CREATE UNIQUE INDEX uq_submission_live
    ON submission (user_id, place_id)
    WHERE deleted_at IS NULL;

-- =========================================================================
-- Captura: contribucion atomica (un hecho + su calidad)
-- =========================================================================
CREATE TABLE contribution (
    id               BIGSERIAL PRIMARY KEY,
    code             UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    -- Siempre cuelga de la valoracion viva del usuario para ese lugar.
    submission_id    BIGINT NOT NULL REFERENCES submission(id) ON DELETE CASCADE,
    -- Denormalizados desde submission (comodidad de queries de agregacion).
    user_id          BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    place_id         BIGINT NOT NULL REFERENCES place(id) ON DELETE CASCADE,
    criterion_id     BIGINT NOT NULL REFERENCES criterion(id),
    answer_option_id BIGINT NOT NULL REFERENCES answer_option(id),
    -- Copiados de answer_option al crear (estables historicamente, no derivados en lectura).
    exists_flag      BOOLEAN,             -- exists de la opcion (NULL = no se)
    quality          SMALLINT CHECK (quality BETWEEN 1 AND 5),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Fecha de validez del dato: se refresca al editar in-place. La agregacion y
    -- la UI leen updated_at (no created_at). No implica decaimiento.
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ          -- soft delete = Deshacer
);

-- Una respuesta viva por criterio dentro de la valoracion de un usuario;
-- re-responder = UPDATE in-place de esa fila.
CREATE UNIQUE INDEX uq_contribution_live
    ON contribution (submission_id, criterion_id)
    WHERE deleted_at IS NULL;

-- Fotos: pertenecen a una submission; opcionalmente evidencian una contribucion concreta.
CREATE TABLE photo (
    id              BIGSERIAL PRIMARY KEY,
    code            UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    submission_id   BIGINT NOT NULL REFERENCES submission(id) ON DELETE CASCADE,
    contribution_id BIGINT REFERENCES contribution(id) ON DELETE SET NULL,
    url             TEXT NOT NULL,
    object_key      TEXT,
    suggested_slot  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- =========================================================================
-- Agregacion materializada: una fila por (place, criterion)
-- Recalculada en la MISMA TX que la contribucion (o su borrado).
-- Estado (verde/amarillo/rojo/gris) y confianza NO se guardan: se derivan.
-- =========================================================================
CREATE TABLE place_criterion_cache (
    place_id             BIGINT NOT NULL REFERENCES place(id) ON DELETE CASCADE,
    criterion_id         BIGINT NOT NULL REFERENCES criterion(id) ON DELETE CASCADE,
    n_yes                INT NOT NULL DEFAULT 0,   -- exists=true
    n_no                 INT NOT NULL DEFAULT 0,   -- exists=false
    n_unsure             INT NOT NULL DEFAULT 0,   -- exists NULL
    quality_p50          NUMERIC,                  -- mediana de quality entre exists=true
    n_photos             INT NOT NULL DEFAULT 0,   -- contribuciones vivas con foto (confianza)
    last_contribution_at TIMESTAMPTZ,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (place_id, criterion_id)
);

-- =========================================================================
-- Colecciones (listas de lugares guardados por el usuario)
-- =========================================================================
CREATE TABLE collection (
    id         BIGSERIAL PRIMARY KEY,
    code       UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE collection_place (
    collection_id BIGINT NOT NULL REFERENCES collection(id) ON DELETE CASCADE,
    place_id      BIGINT NOT NULL REFERENCES place(id) ON DELETE CASCADE,
    added_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (collection_id, place_id)
);

-- =========================================================================
-- Google Maps API call log (quota tracking)
-- =========================================================================
CREATE TABLE gmaps_api_log (
    id        BIGSERIAL PRIMARY KEY,
    called_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- Indices
-- =========================================================================
CREATE INDEX idx_place_created_by         ON place(created_by);
CREATE INDEX idx_place_location           ON place(latitude, longitude);
CREATE INDEX idx_place_deleted_at         ON place(deleted_at);
CREATE INDEX idx_place_google_place_id    ON place(google_place_id) WHERE google_place_id IS NOT NULL;
CREATE INDEX idx_place_published_location  ON place(latitude, longitude) WHERE published AND deleted_at IS NULL;

CREATE INDEX idx_criterion_dimension      ON criterion(dimension_id);
CREATE INDEX idx_criterion_depends_on     ON criterion(depends_on_criterion_id) WHERE depends_on_criterion_id IS NOT NULL;
CREATE INDEX idx_answer_option_criterion  ON answer_option(criterion_id);

CREATE INDEX idx_contribution_place       ON contribution(place_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_contribution_user        ON contribution(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_contribution_criterion   ON contribution(criterion_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_contribution_submission  ON contribution(submission_id) WHERE submission_id IS NOT NULL;

CREATE INDEX idx_pcc_place                ON place_criterion_cache(place_id);

CREATE INDEX idx_submission_place         ON submission(place_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_submission_user          ON submission(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_photo_submission         ON photo(submission_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_photo_contribution       ON photo(contribution_id) WHERE contribution_id IS NOT NULL;

CREATE INDEX idx_collection_user_id       ON collection(user_id);
CREATE INDEX idx_user_profile_need_user   ON user_profile_need(user_id);
CREATE INDEX idx_gmaps_log_called_at      ON gmaps_api_log(called_at);
