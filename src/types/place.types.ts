export interface Place {
  id: number;
  name: string;
  description?: string;
  address: string;
  city: string;
  state?: string;
  country: string;
  postal_code?: string;
  latitude: number;
  longitude: number;
  phone?: string;
  website?: string;
  category?: string;
  overall_accessibility_rating: number;
  created_by?: number;
  created_at: Date;
  updated_at: Date;
}

export interface PlaceWithRatings extends Place {
  avg_rating: number;
  review_count: number;
}

export interface PlaceCreateInput {
  name: string;
  description?: string;
  address: string;
  city: string;
  state?: string;
  country: string;
  postal_code?: string;
  latitude: number;
  longitude: number;
  phone?: string;
  website?: string;
  category?: string;
  created_by?: number;
}

export interface PlaceUpdateInput {
  name?: string;
  description?: string;
  address?: string;
  city?: string;
  state?: string;
  country?: string;
  postal_code?: string;
  latitude?: number;
  longitude?: number;
  phone?: string;
  website?: string;
  category?: string;
}

export interface PlaceSearchFilters {
  city?: string;
  category?: string;
  latitude?: number;
  longitude?: number;
  radius?: number;
  limit?: number;
  offset?: number;
}

export interface PlaceAccessibilityFeature {
  place_id: number;
  feature_id: number;
  is_available: boolean;
  notes?: string;
  created_at: Date;
}

export interface AddFeatureInput {
  feature_id: number;
  is_available?: boolean;
  notes?: string;
}
