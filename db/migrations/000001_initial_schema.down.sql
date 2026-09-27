-- Down migration 000001: revierte el schema inicial.
-- Idempotente: usar DROP IF EXISTS para evitar errores si se re-ejecuta.
-- Orden: tablas con FK primero (las hijas antes que las madres).

DROP TABLE IF EXISTS accesspath.gmaps_api_log CASCADE;
DROP TABLE IF EXISTS accesspath.user_profile_need CASCADE;
DROP TABLE IF EXISTS accesspath.collection_place CASCADE;
DROP TABLE IF EXISTS accesspath.collection CASCADE;
DROP TABLE IF EXISTS accesspath.photo CASCADE;
DROP TABLE IF EXISTS accesspath.submission CASCADE;
DROP TABLE IF EXISTS accesspath.contribution CASCADE;
DROP TABLE IF EXISTS accesspath.place_criterion_cache CASCADE;
DROP TABLE IF EXISTS accesspath.criterion CASCADE;
DROP TABLE IF EXISTS accesspath.dimension CASCADE;
DROP TABLE IF EXISTS accesspath.place CASCADE;
DROP TABLE IF EXISTS accesspath."user" CASCADE;

-- El esquema accesspath se queda intencionalmente: si otras tablas fuera
-- del scope de este repo lo usan, no las eliminamos.
