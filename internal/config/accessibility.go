package config

import (
	"encoding/json"
	"log"
	"os"
)

// AccessibilitySelection agrupa los factores del algoritmo de seleccion de pregunta.
type AccessibilitySelection struct {
	RelevanceMatch   float64 `json:"relevance_match"`
	RelevanceNoMatch float64 `json:"relevance_no_match"`
	ConflictBonus    float64 `json:"conflict_bonus"`
	TopK             int     `json:"top_k"`
}

// AccessibilityThresholds son los umbrales de agregacion y seleccion.
// Se cargan de un archivo JSON externo (ACCESSIBILITY_CONFIG_PATH) para poder
// ajustarlos sin recompilar. Nunca hardcodear estos valores en la logica.
type AccessibilityThresholds struct {
	ConfidenceMin int                    `json:"confidence_min"`
	ConsensoSi    float64                `json:"consenso_si"`
	ConsensoNo    float64                `json:"consenso_no"`
	CalBuena      float64                `json:"cal_buena"`
	CalRegular    float64                `json:"cal_regular"`
	Selection     AccessibilitySelection `json:"selection"`
}

// defaultAccessibilityThresholds son los valores de arranque documentados en el
// prompt de rework, usados si el archivo externo no existe o no se puede leer.
func defaultAccessibilityThresholds() AccessibilityThresholds {
	return AccessibilityThresholds{
		ConfidenceMin: 1,
		ConsensoSi:    0.60,
		ConsensoNo:    0.40,
		CalBuena:      4.0,
		CalRegular:    2.5,
		Selection: AccessibilitySelection{
			RelevanceMatch:   1.0,
			RelevanceNoMatch: 0.4,
			ConflictBonus:    0.5,
			TopK:             3,
		},
	}
}

// LoadAccessibilityThresholds lee el archivo JSON de umbrales. Si no existe o
// falla el parseo, devuelve los valores por defecto y avisa por log (sin abortar).
func LoadAccessibilityThresholds(path string) *AccessibilityThresholds {
	def := defaultAccessibilityThresholds()

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("umbrales de accesibilidad: no se pudo leer %q, usando valores por defecto: %v", path, err)
		return &def
	}

	cfg := def
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("umbrales de accesibilidad: JSON invalido en %q, usando valores por defecto: %v", path, err)
		return &def
	}
	log.Printf("umbrales de accesibilidad cargados de %q", path)
	return &cfg
}
