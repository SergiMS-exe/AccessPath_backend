import pool from '../config/database';
import bcrypt from 'bcryptjs';
import { User, UserCreateInput, UserResponse } from '../types/user.types';

class UserModel {
  static async create({ email, password, full_name }: UserCreateInput): Promise<UserResponse> {
    const password_hash = await bcrypt.hash(password, 10);

    const result = await pool.query<UserResponse>(
      'INSERT INTO users (email, password_hash, full_name) VALUES ($1, $2, $3) RETURNING id, email, full_name, created_at',
      [email, password_hash, full_name]
    );

    return result.rows[0];
  }

  static async findByEmail(email: string): Promise<User | undefined> {
    const result = await pool.query<User>(
      'SELECT * FROM users WHERE email = $1',
      [email]
    );

    return result.rows[0];
  }

  static async findById(id: number): Promise<UserResponse | undefined> {
    const result = await pool.query<UserResponse>(
      'SELECT id, email, full_name, created_at, updated_at FROM users WHERE id = $1',
      [id]
    );

    return result.rows[0];
  }

  static async verifyPassword(plainPassword: string, hashedPassword: string): Promise<boolean> {
    return await bcrypt.compare(plainPassword, hashedPassword);
  }
}

export default UserModel;
