import pool from '../config/database';
import { Review, ReviewWithUser, ReviewCreateInput, ReviewUpdateInput } from '../types/review.types';

class ReviewModel {
  static async create({ place_id, user_id, rating, comment }: ReviewCreateInput): Promise<Review> {
    const result = await pool.query<Review>(
      `INSERT INTO reviews (place_id, user_id, rating, comment)
       VALUES ($1, $2, $3, $4)
       RETURNING *`,
      [place_id, user_id, rating, comment]
    );

    return result.rows[0];
  }

  static async findById(id: number): Promise<ReviewWithUser | undefined> {
    const result = await pool.query<ReviewWithUser>(
      `SELECT r.*, u.full_name as user_name, u.email as user_email
       FROM reviews r
       LEFT JOIN users u ON r.user_id = u.id
       WHERE r.id = $1`,
      [id]
    );

    return result.rows[0];
  }

  static async findByPlaceId(placeId: number): Promise<ReviewWithUser[]> {
    const result = await pool.query<ReviewWithUser>(
      `SELECT r.*, u.full_name as user_name
       FROM reviews r
       LEFT JOIN users u ON r.user_id = u.id
       WHERE r.place_id = $1
       ORDER BY r.created_at DESC`,
      [placeId]
    );

    return result.rows;
  }

  static async update(id: number, { rating, comment }: ReviewUpdateInput): Promise<Review | undefined> {
    const result = await pool.query<Review>(
      `UPDATE reviews
       SET rating = $1, comment = $2, updated_at = CURRENT_TIMESTAMP
       WHERE id = $3
       RETURNING *`,
      [rating, comment, id]
    );

    return result.rows[0];
  }

  static async delete(id: number): Promise<{ id: number } | undefined> {
    const result = await pool.query<{ id: number }>(
      'DELETE FROM reviews WHERE id = $1 RETURNING id',
      [id]
    );
    return result.rows[0];
  }
}

export default ReviewModel;
