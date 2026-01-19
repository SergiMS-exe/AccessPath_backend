export interface JwtPayload {
  userId: number;
  iat?: number;
  exp?: number;
}

export interface AuthResponse {
  message: string;
  user: {
    id: number;
    email: string;
    full_name: string;
  };
  token: string;
}
