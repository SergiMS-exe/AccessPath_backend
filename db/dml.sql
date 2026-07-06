-- DML (Data Manipulation Language) - Seed del catalogo AccessPath v2
--
-- Reescrito desde cero. Siembra dimensiones -> criterios -> opciones de respuesta.
-- Cada opcion codifica el hecho (exists) y su calidad (quality).
-- Idempotente: dimensiones/criterios por ON CONFLICT(key); opciones se saltan si
-- el criterio ya tiene opciones (guardas NOT EXISTS).
--
-- Notacion del prompt de rework: [S]=starter, [B]=blocking, dep:=depends_on.
-- Los textos de prompt/label son UI y pueden llevar acentos.

SET search_path TO accesspath, public;

-- =========================================================================
-- Dimensiones
-- =========================================================================
-- Completas: acceso, aseos, sensorial_env, personal.
-- Ligeras: ceguera, auditiva.
-- Declaradas sin criterios (para sembrar luego sin migrar): movilidad_interior,
-- orientacion, cognitiva, info_previa.
INSERT INTO dimension (key, name, description, sort_order) VALUES
    ('acceso',              'Llegada y acceso',        'Aproximacion y entrada al lugar.',                 1),
    ('aseos',               'Aseos adaptados',         'Existencia y usabilidad de aseo adaptado.',        2),
    ('sensorial_env',       'Entorno sensorial',       'Ruido, luz, aglomeracion y espacios de calma.',    3),
    ('personal',            'Personal y servicio',     'Actitud y formacion del personal en accesibilidad.', 4),
    ('ceguera',             'Acceso para ceguera',     'Orientacion tactil y perro guia.',                 5),
    ('auditiva',            'Comunicacion auditiva',   'Bucle magnetico y avisos visuales.',               6),
    ('movilidad_interior',  'Movilidad interior',      'Circulacion dentro del local.',                    7),
    ('orientacion',         'Orientacion y senalizacion', 'Facilidad para orientarse.',                    8),
    ('cognitiva',           'Accesibilidad cognitiva', 'Lectura facil, pictogramas, previsibilidad.',      9),
    ('info_previa',         'Informacion previa',      'Datos de accesibilidad disponibles antes de ir.',  10)
ON CONFLICT (key) DO NOTHING;

-- =========================================================================
-- Criterios
-- =========================================================================
-- acceso ------------------------------------------------------------------
INSERT INTO criterion (dimension_id, key, prompt, profile_tags, is_blocking, is_starter, sort_order)
SELECT id, 'acceso.entrada', '¿Cómo es la entrada al lugar?',
       ARRAY['silla','movilidad_reducida','baja_vision'], TRUE, TRUE, 1
FROM dimension WHERE key = 'acceso'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'acceso.puerta', '¿La puerta de entrada deja pasar una silla de ruedas?',
       ARRAY['silla'], 2
FROM dimension WHERE key = 'acceso'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'acceso.aparcamiento', '¿Hay aparcamiento accesible cerca?',
       ARRAY['silla','movilidad_reducida'], 3
FROM dimension WHERE key = 'acceso'
ON CONFLICT (key) DO NOTHING;

-- aseos -------------------------------------------------------------------
INSERT INTO criterion (dimension_id, key, prompt, profile_tags, is_blocking, sort_order)
SELECT id, 'aseos.existe', '¿Hay un aseo adaptado?',
       ARRAY['silla','movilidad_reducida'], TRUE, 1
FROM dimension WHERE key = 'aseos'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, depends_on_criterion_id, sort_order)
SELECT d.id, 'aseos.disponible', '¿El aseo adaptado está disponible y usable?',
       ARRAY['silla','movilidad_reducida'], (SELECT id FROM criterion WHERE key = 'aseos.existe'), 2
FROM dimension d WHERE d.key = 'aseos'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, depends_on_criterion_id, sort_order)
SELECT d.id, 'aseos.espacio_giro', '¿Hay espacio para girar una silla dentro del aseo?',
       ARRAY['silla'], (SELECT id FROM criterion WHERE key = 'aseos.existe'), 3
FROM dimension d WHERE d.key = 'aseos'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, depends_on_criterion_id, sort_order)
SELECT d.id, 'aseos.barras', '¿El aseo tiene barras de apoyo?',
       ARRAY['silla','movilidad_reducida'], (SELECT id FROM criterion WHERE key = 'aseos.existe'), 4
FROM dimension d WHERE d.key = 'aseos'
ON CONFLICT (key) DO NOTHING;

-- sensorial_env -----------------------------------------------------------
INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'sensorial_env.ruido', '¿Cómo es el nivel de ruido?',
       ARRAY['sensorial','auditiva'], 1
FROM dimension WHERE key = 'sensorial_env'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'sensorial_env.luz', '¿Cómo es la iluminación?',
       ARRAY['sensorial','baja_vision'], 2
FROM dimension WHERE key = 'sensorial_env'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'sensorial_env.espacio_tranquilo', '¿Hay un espacio tranquilo donde retirarse?',
       ARRAY['sensorial','cognitiva'], 3
FROM dimension WHERE key = 'sensorial_env'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'sensorial_env.aglomeracion', '¿Cómo suele estar de gente?',
       ARRAY['sensorial'], 4
FROM dimension WHERE key = 'sensorial_env'
ON CONFLICT (key) DO NOTHING;

-- personal (transversal) --------------------------------------------------
INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'personal.ayuda', '¿El personal te ayudó cuando lo necesitaste?',
       ARRAY['silla','movilidad_reducida','baja_vision','ceguera','auditiva','cognitiva','sensorial'], 1
FROM dimension WHERE key = 'personal'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, sort_order)
SELECT id, 'personal.formado', '¿El personal parecía formado en accesibilidad?', 2
FROM dimension WHERE key = 'personal'
ON CONFLICT (key) DO NOTHING;

-- ceguera (ligera) --------------------------------------------------------
INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'ceguera.perro_guia', '¿Se permite la entrada del perro guía?',
       ARRAY['ceguera'], 1
FROM dimension WHERE key = 'ceguera'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'ceguera.pavimento_tactil', '¿Hay pavimento táctil para orientarse?',
       ARRAY['ceguera','baja_vision'], 2
FROM dimension WHERE key = 'ceguera'
ON CONFLICT (key) DO NOTHING;

-- auditiva (ligera) -------------------------------------------------------
INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'auditiva.bucle', '¿Hay bucle magnético para audífonos?',
       ARRAY['auditiva'], 1
FROM dimension WHERE key = 'auditiva'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'auditiva.avisos_visuales', '¿Los avisos importantes también se muestran de forma visual?',
       ARRAY['auditiva'], 2
FROM dimension WHERE key = 'auditiva'
ON CONFLICT (key) DO NOTHING;

-- movilidad_interior (alto valor para silla) -----------------------------
INSERT INTO criterion (dimension_id, key, prompt, profile_tags, is_blocking, sort_order)
SELECT id, 'movilidad_interior.circulacion', '¿Se puede circular por dentro con una silla de ruedas?',
       ARRAY['silla','movilidad_reducida'], TRUE, 1
FROM dimension WHERE key = 'movilidad_interior'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, sort_order)
SELECT id, 'movilidad_interior.desnivel', '¿Hay escalones o desniveles dentro del local?',
       ARRAY['silla','movilidad_reducida'], 2
FROM dimension WHERE key = 'movilidad_interior'
ON CONFLICT (key) DO NOTHING;

INSERT INTO criterion (dimension_id, key, prompt, profile_tags, depends_on_criterion_id, sort_order)
SELECT d.id, 'movilidad_interior.ascensor', '¿Hay ascensor o plataforma entre plantas?',
       ARRAY['silla','movilidad_reducida'], (SELECT id FROM criterion WHERE key = 'movilidad_interior.desnivel'), 3
FROM dimension d WHERE d.key = 'movilidad_interior'
ON CONFLICT (key) DO NOTHING;

-- =========================================================================
-- Opciones de respuesta (exists / quality codificados en la opcion)
-- Una sentencia por criterio; se salta si el criterio ya tiene opciones.
-- =========================================================================
-- acceso.entrada
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Entra sin escalón'::text,        TRUE,  5::smallint, FALSE, 1),
    ('Hay rampa y es cómoda',          TRUE,  5,           FALSE, 2),
    ('Hay rampa pero es difícil',      TRUE,  2,           FALSE, 3),
    ('Tiene un escalón',               FALSE, NULL,        FALSE, 4),
    ('No lo sé',                       NULL,  NULL,        TRUE,  5)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'acceso.entrada'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- acceso.puerta
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Pasa una silla con holgura'::text, TRUE,  5::smallint, FALSE, 1),
    ('Pasa justa',                       TRUE,  2,           FALSE, 2),
    ('No pasa una silla',                FALSE, NULL,        FALSE, 3),
    ('No lo sé',                         NULL,  NULL,        TRUE,  4)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'acceso.puerta'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- acceso.aparcamiento
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, reservado y cerca'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('No hay',                      FALSE, NULL,           FALSE, 2),
    ('No lo sé',                    NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'acceso.aparcamiento'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- aseos.existe
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, hay aseo adaptado'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('No hay',                      FALSE, NULL,           FALSE, 2),
    ('No lo sé',                    NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'aseos.existe'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- aseos.disponible
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, disponible y usable'::text,    TRUE,  NULL::smallint, FALSE, 1),
    ('No, cerrado o usado de almacén',   FALSE, NULL,           FALSE, 2),
    ('No lo sé',                         NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'aseos.disponible'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- aseos.espacio_giro
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Cabe girar con holgura'::text, TRUE,  5::smallint, FALSE, 1),
    ('Justo',                        TRUE,  2,           FALSE, 2),
    ('No cabe el giro',              FALSE, NULL,        FALSE, 3),
    ('No lo sé',                     NULL,  NULL,        TRUE,  4)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'aseos.espacio_giro'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- aseos.barras
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, tiene barras de apoyo'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('No tiene',                        FALSE, NULL,           FALSE, 2),
    ('No lo sé',                        NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'aseos.barras'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- sensorial_env.ruido
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Tranquilo'::text,  TRUE, 5::smallint, FALSE, 1),
    ('Aceptable',        TRUE, 3,           FALSE, 2),
    ('Muy ruidoso',      TRUE, 1,           FALSE, 3),
    ('No lo sé',         NULL, NULL,        TRUE,  4)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'sensorial_env.ruido'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- sensorial_env.luz
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Luz natural o cálida'::text,       TRUE, 5::smallint, FALSE, 1),
    ('Fluorescente intensa',             TRUE, 2,           FALSE, 2),
    ('Parpadeante o deslumbrante',       TRUE, 1,           FALSE, 3),
    ('No lo sé',                         NULL, NULL,        TRUE,  4)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'sensorial_env.luz'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- sensorial_env.espacio_tranquilo
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, hay un rincón tranquilo'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('No',                                FALSE, NULL,           FALSE, 2),
    ('No lo sé',                          NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'sensorial_env.espacio_tranquilo'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- sensorial_env.aglomeracion
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Tranquilo y previsible'::text, TRUE, 5::smallint, FALSE, 1),
    ('Suele llenarse',               TRUE, 2,           FALSE, 2),
    ('No lo sé',                     NULL, NULL,        TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'sensorial_env.aglomeracion'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- personal.ayuda
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, el personal ayudó'::text, TRUE, 5::smallint, FALSE, 1),
    ('Ayuda regular',               TRUE, 3,           FALSE, 2),
    ('No ayudó / no supo',          TRUE, 1,           FALSE, 3),
    ('No lo sé',                    NULL, NULL,        TRUE,  4)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'personal.ayuda'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- personal.formado
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Parecía cómodo con accesibilidad'::text, TRUE, 5::smallint, FALSE, 1),
    ('No mucho',                               TRUE, 2,           FALSE, 2),
    ('No lo sé',                               NULL, NULL,        TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'personal.formado'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- ceguera.perro_guia
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, se permite el perro guía'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('No',                                 FALSE, NULL,           FALSE, 2),
    ('No lo sé',                           NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'ceguera.perro_guia'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- ceguera.pavimento_tactil
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, hay pavimento táctil'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('No',                             FALSE, NULL,           FALSE, 2),
    ('No lo sé',                       NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'ceguera.pavimento_tactil'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- auditiva.bucle
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, hay bucle magnético señalizado'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('No',                                       FALSE, NULL,           FALSE, 2),
    ('No lo sé',                                 NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'auditiva.bucle'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- auditiva.avisos_visuales
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, los avisos también son visuales'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('No, solo por voz',                          FALSE, NULL,           FALSE, 2),
    ('No lo sé',                                  NULL,  NULL,           TRUE,  3)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'auditiva.avisos_visuales'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- movilidad_interior.circulacion
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Se circula con holgura'::text, TRUE,  5::smallint, FALSE, 1),
    ('Se pasa justo',                TRUE,  2,           FALSE, 2),
    ('No se puede circular',         FALSE, NULL,        FALSE, 3),
    ('No lo sé',                     NULL,  NULL,        TRUE,  4)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'movilidad_interior.circulacion'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- movilidad_interior.desnivel (exists=true => SI hay desnivel; habilita ascensor)
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Todo a un mismo nivel'::text,        FALSE, NULL::smallint, FALSE, 1),
    ('Hay algún desnivel salvable',        TRUE,  2,              FALSE, 2),
    ('Hay escalones que no se pueden salvar', TRUE, 1,            FALSE, 3),
    ('No lo sé',                           NULL,  NULL,           TRUE,  4)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'movilidad_interior.desnivel'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);

-- movilidad_interior.ascensor
INSERT INTO answer_option (criterion_id, label, exists_value, quality_value, is_unsure, sort_order)
SELECT c.id, v.label, v.ev, v.qv, v.un, v.so
FROM criterion c CROSS JOIN (VALUES
    ('Sí, hay ascensor o plataforma y funciona'::text, TRUE,  NULL::smallint, FALSE, 1),
    ('Hay pero no siempre funciona',                   TRUE,  2,              FALSE, 2),
    ('No hay',                                         FALSE, NULL,           FALSE, 3),
    ('No lo sé',                                       NULL,  NULL,           TRUE,  4)
) AS v(label, ev, qv, un, so)
WHERE c.key = 'movilidad_interior.ascensor'
  AND NOT EXISTS (SELECT 1 FROM answer_option ao WHERE ao.criterion_id = c.id);
