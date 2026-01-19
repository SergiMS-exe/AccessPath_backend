import pool from '../config/database';
import { AccessibilityFeature } from '../types/feature.types';

class FeatureModel {
  static async findAll(): Promise<AccessibilityFeature[]> {
    const result = await pool.query<AccessibilityFeature>(
      'SELECT * FROM accessibility_features ORDER BY category, name'
    );

    return result.rows;
  }

  static async findById(id: number): Promise<AccessibilityFeature | undefined> {
    const result = await pool.query<AccessibilityFeature>(
      'SELECT * FROM accessibility_features WHERE id = $1',
      [id]
    );

    return result.rows[0];
  }

  static async findByCategory(category: string): Promise<AccessibilityFeature[]> {
    const result = await pool.query<AccessibilityFeature>(
      'SELECT * FROM accessibility_features WHERE category = $1 ORDER BY name',
      [category]
    );

    return result.rows;
  }
}

export default FeatureModel;
