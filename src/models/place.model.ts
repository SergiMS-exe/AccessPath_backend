import pool from '../config/database';
import {
  Place,
  PlaceWithRatings,
  PlaceCreateInput,
  PlaceUpdateInput,
  PlaceSearchFilters,
  PlaceAccessibilityFeature,
} from '../types/place.types';
import { FeatureWithAvailability } from '../types/feature.types';

class PlaceModel {
  static async create(placeData: PlaceCreateInput): Promise<Place> {
    const {
      name, description, address, city, state, country, postal_code,
      latitude, longitude, phone, website, category, created_by
    } = placeData;

    const result = await pool.query<Place>(
      `INSERT INTO places
       (name, description, address, city, state, country, postal_code, latitude, longitude, phone, website, category, created_by)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
       RETURNING *`,
      [name, description, address, city, state, country, postal_code, latitude, longitude, phone, website, category, created_by]
    );

    return result.rows[0];
  }

  static async findById(id: number): Promise<PlaceWithRatings | undefined> {
    const result = await pool.query<PlaceWithRatings>(
      `SELECT p.*,
              COALESCE(AVG(r.rating), 0) as avg_rating,
              COUNT(DISTINCT r.id) as review_count
       FROM places p
       LEFT JOIN reviews r ON p.id = r.place_id
       WHERE p.id = $1
       GROUP BY p.id`,
      [id]
    );

    return result.rows[0];
  }

  static async findAll(filters: PlaceSearchFilters = {}): Promise<PlaceWithRatings[]> {
    let query = `
      SELECT p.*,
             COALESCE(AVG(r.rating), 0) as avg_rating,
             COUNT(DISTINCT r.id) as review_count
      FROM places p
      LEFT JOIN reviews r ON p.id = r.place_id
      WHERE 1=1
    `;

    const params: any[] = [];
    let paramIndex = 1;

    if (filters.city) {
      query += ` AND LOWER(p.city) = LOWER($${paramIndex})`;
      params.push(filters.city);
      paramIndex++;
    }

    if (filters.category) {
      query += ` AND p.category = $${paramIndex}`;
      params.push(filters.category);
      paramIndex++;
    }

    if (filters.latitude && filters.longitude && filters.radius) {
      query += ` AND (
        6371 * acos(
          cos(radians($${paramIndex})) * cos(radians(p.latitude)) *
          cos(radians(p.longitude) - radians($${paramIndex + 1})) +
          sin(radians($${paramIndex})) * sin(radians(p.latitude))
        )
      ) <= $${paramIndex + 2}`;
      params.push(filters.latitude, filters.longitude, filters.radius);
      paramIndex += 3;
    }

    query += ` GROUP BY p.id ORDER BY p.created_at DESC`;

    if (filters.limit) {
      query += ` LIMIT $${paramIndex}`;
      params.push(filters.limit);
      paramIndex++;
    }

    if (filters.offset) {
      query += ` OFFSET $${paramIndex}`;
      params.push(filters.offset);
    }

    const result = await pool.query<PlaceWithRatings>(query, params);
    return result.rows;
  }

  static async update(id: number, placeData: PlaceUpdateInput): Promise<Place | undefined> {
    const fields: string[] = [];
    const values: any[] = [];
    let paramIndex = 1;

    Object.keys(placeData).forEach(key => {
      if ((placeData as any)[key] !== undefined) {
        fields.push(`${key} = $${paramIndex}`);
        values.push((placeData as any)[key]);
        paramIndex++;
      }
    });

    if (fields.length === 0) {
      return this.findById(id);
    }

    fields.push(`updated_at = CURRENT_TIMESTAMP`);
    values.push(id);

    const query = `UPDATE places SET ${fields.join(', ')} WHERE id = $${paramIndex} RETURNING *`;
    const result = await pool.query<Place>(query, values);

    return result.rows[0];
  }

  static async delete(id: number): Promise<{ id: number } | undefined> {
    const result = await pool.query<{ id: number }>(
      'DELETE FROM places WHERE id = $1 RETURNING id',
      [id]
    );
    return result.rows[0];
  }

  static async addFeature(
    placeId: number,
    featureId: number,
    isAvailable: boolean = true,
    notes: string | null = null
  ): Promise<PlaceAccessibilityFeature> {
    const result = await pool.query<PlaceAccessibilityFeature>(
      `INSERT INTO place_accessibility_features (place_id, feature_id, is_available, notes)
       VALUES ($1, $2, $3, $4)
       ON CONFLICT (place_id, feature_id)
       DO UPDATE SET is_available = $3, notes = $4
       RETURNING *`,
      [placeId, featureId, isAvailable, notes]
    );

    return result.rows[0];
  }

  static async getFeatures(placeId: number): Promise<FeatureWithAvailability[]> {
    const result = await pool.query<FeatureWithAvailability>(
      `SELECT af.*, paf.is_available, paf.notes
       FROM accessibility_features af
       JOIN place_accessibility_features paf ON af.id = paf.feature_id
       WHERE paf.place_id = $1`,
      [placeId]
    );

    return result.rows;
  }

  static async removeFeature(placeId: number, featureId: number): Promise<PlaceAccessibilityFeature | undefined> {
    const result = await pool.query<PlaceAccessibilityFeature>(
      'DELETE FROM place_accessibility_features WHERE place_id = $1 AND feature_id = $2 RETURNING *',
      [placeId, featureId]
    );

    return result.rows[0];
  }
}

export default PlaceModel;
