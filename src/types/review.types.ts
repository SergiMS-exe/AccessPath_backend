export interface Review {
  id: number;
  place_id: number;
  user_id?: number;
  rating: number;
  comment?: string;
  created_at: Date;
  updated_at: Date;
}

export interface ReviewWithUser extends Review {
  user_name?: string;
  user_email?: string;
}

export interface ReviewCreateInput {
  place_id: number;
  user_id: number;
  rating: number;
  comment?: string;
}

export interface ReviewUpdateInput {
  rating: number;
  comment?: string;
}
