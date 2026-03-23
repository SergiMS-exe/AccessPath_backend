-- DML (Data Manipulation Language) - Initial seed data

-- Categories
INSERT INTO categories (name, slug, is_active, display_order) VALUES
    ('Física',    'fisica',    TRUE, 1),
    ('Sensorial', 'sensorial', TRUE, 2),
    ('Psíquica',  'psiquica',  TRUE, 3)
ON CONFLICT (slug) DO NOTHING;

-- Subcategories - Física
INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Rampa de acceso',    'rampa-de-acceso',    TRUE, 1 FROM categories WHERE slug = 'fisica'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Ascensor adaptado',  'ascensor-adaptado',  TRUE, 2 FROM categories WHERE slug = 'fisica'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Baño adaptado',      'bano-adaptado',      TRUE, 3 FROM categories WHERE slug = 'fisica'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Parking accesible',  'parking-accesible',  TRUE, 4 FROM categories WHERE slug = 'fisica'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Puertas anchas',     'puertas-anchas',     TRUE, 5 FROM categories WHERE slug = 'fisica'
ON CONFLICT (slug) DO NOTHING;

-- Subcategories - Sensorial
INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Señalización Braille',         'senalizacion-braille',          TRUE, 1 FROM categories WHERE slug = 'sensorial'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Bucle magnético',              'bucle-magnetico',               TRUE, 2 FROM categories WHERE slug = 'sensorial'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Señalización visual',          'senalizacion-visual',           TRUE, 3 FROM categories WHERE slug = 'sensorial'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Personal con lengua de signos','personal-lengua-signos',        TRUE, 4 FROM categories WHERE slug = 'sensorial'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Iluminación adecuada',         'iluminacion-adecuada',          TRUE, 5 FROM categories WHERE slug = 'sensorial'
ON CONFLICT (slug) DO NOTHING;

-- Subcategories - Psíquica
INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Señalización clara',           'senalizacion-clara',            TRUE, 1 FROM categories WHERE slug = 'psiquica'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Espacio tranquilo',            'espacio-tranquilo',             TRUE, 2 FROM categories WHERE slug = 'psiquica'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Personal formado',             'personal-formado',              TRUE, 3 FROM categories WHERE slug = 'psiquica'
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
SELECT id, 'Información en lectura fácil', 'informacion-lectura-facil',     TRUE, 4 FROM categories WHERE slug = 'psiquica'
ON CONFLICT (slug) DO NOTHING;
