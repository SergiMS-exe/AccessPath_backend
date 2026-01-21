-- DML (Data Manipulation Language) - Initial seed data

-- Insert fixed categories
INSERT INTO feature_categories (name, description, icon) VALUES
    ('Física', 'Accesibilidad física y movilidad', 'wheelchair'),
    ('Sensorial', 'Accesibilidad sensorial (visual y auditiva)', 'eye'),
    ('Psíquica', 'Accesibilidad cognitiva y psíquica', 'brain')
ON CONFLICT (name) DO NOTHING;

-- Insert features for Física category
INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Rampa de acceso', 'Rampa para acceso con silla de ruedas', 'ramp'
FROM feature_categories WHERE name = 'Física'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Ascensor adaptado', 'Ascensor con espacio para silla de ruedas', 'elevator'
FROM feature_categories WHERE name = 'Física'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Baño adaptado', 'Baño con barras de apoyo y espacio suficiente', 'toilet'
FROM feature_categories WHERE name = 'Física'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Parking accesible', 'Plazas de aparcamiento reservadas', 'parking'
FROM feature_categories WHERE name = 'Física'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Puertas anchas', 'Puertas con ancho suficiente para silla de ruedas', 'door'
FROM feature_categories WHERE name = 'Física'
ON CONFLICT DO NOTHING;

-- Insert features for Sensorial category
INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Señalización Braille', 'Carteles y señales en Braille', 'braille'
FROM feature_categories WHERE name = 'Sensorial'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Bucle magnético', 'Sistema de inducción magnética para audífonos', 'hearing'
FROM feature_categories WHERE name = 'Sensorial'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Señalización visual', 'Alertas visuales además de sonoras', 'alert'
FROM feature_categories WHERE name = 'Sensorial'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Personal con lengua de signos', 'Personal que conoce lengua de signos', 'sign-language'
FROM feature_categories WHERE name = 'Sensorial'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Iluminación adecuada', 'Buena iluminación para personas con baja visión', 'light'
FROM feature_categories WHERE name = 'Sensorial'
ON CONFLICT DO NOTHING;

-- Insert features for Psíquica category
INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Señalización clara', 'Señalización con pictogramas y texto simple', 'signpost'
FROM feature_categories WHERE name = 'Psíquica'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Espacio tranquilo', 'Zona de descanso con poco ruido', 'quiet'
FROM feature_categories WHERE name = 'Psíquica'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Personal formado', 'Personal con formación en atención a diversidad funcional', 'support'
FROM feature_categories WHERE name = 'Psíquica'
ON CONFLICT DO NOTHING;

INSERT INTO accessibility_features (category_id, name, description, icon)
SELECT id, 'Información en lectura fácil', 'Documentación adaptada a lectura fácil', 'document'
FROM feature_categories WHERE name = 'Psíquica'
ON CONFLICT DO NOTHING;
