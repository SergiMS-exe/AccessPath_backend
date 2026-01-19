export interface AccessibilityFeature {
  id: number;
  name: string;
  description?: string;
  icon?: string;
  category: string;
  created_at: Date;
}

export interface FeatureWithAvailability extends AccessibilityFeature {
  is_available: boolean;
  notes?: string;
}
