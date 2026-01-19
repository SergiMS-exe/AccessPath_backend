export interface User {
  id: number;
  email: string;
  password_hash: string;
  full_name: string;
  created_at: Date;
  updated_at: Date;
}

export interface UserCreateInput {
  email: string;
  password: string;
  full_name: string;
}

export interface UserResponse {
  id: number;
  email: string;
  full_name: string;
  created_at?: Date;
  updated_at?: Date;
}

export interface LoginInput {
  email: string;
  password: string;
}

export interface RegisterInput {
  email: string;
  password: string;
  full_name: string;
}
