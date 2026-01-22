package domain

import (
	"encoding/json"
)

// SupportedLanguages defines the languages supported by the system
var SupportedLanguages = []string{"en", "ja"}

// DefaultLanguage is the default language when none is specified
const DefaultLanguage = "ja"

// LocalizedText represents text with multiple language versions
type LocalizedText struct {
	EN string `json:"en"`
	JA string `json:"ja"`
}

// Get returns the text for the specified language with fallback
func (t LocalizedText) Get(lang string) string {
	switch lang {
	case "ja":
		if t.JA != "" {
			return t.JA
		}
		return t.EN
	case "en":
		if t.EN != "" {
			return t.EN
		}
		return t.JA
	default:
		// Default to Japanese, then English
		if t.JA != "" {
			return t.JA
		}
		return t.EN
	}
}

// IsEmpty returns true if both language versions are empty
func (t LocalizedText) IsEmpty() bool {
	return t.EN == "" && t.JA == ""
}

// MarshalJSON implements json.Marshaler
func (t LocalizedText) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"en": t.EN,
		"ja": t.JA,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (t *LocalizedText) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	t.EN = m["en"]
	t.JA = m["ja"]
	return nil
}

// NewLocalizedText creates a new LocalizedText with the same value for all languages
func NewLocalizedText(text string) LocalizedText {
	return LocalizedText{EN: text, JA: text}
}

// L is a shorthand for creating LocalizedText
func L(en, ja string) LocalizedText {
	return LocalizedText{EN: en, JA: ja}
}

// LocalizedOutputPort represents an output port with localized labels
type LocalizedOutputPort struct {
	Name        string          `json:"name"`
	Label       LocalizedText   `json:"label"`
	Description LocalizedText   `json:"description,omitempty"`
	IsDefault   bool            `json:"is_default"`
	Schema      json.RawMessage `json:"schema,omitempty"`
}

// ToOutputPort converts to OutputPort for the specified language
func (p LocalizedOutputPort) ToOutputPort(lang string) OutputPort {
	return OutputPort{
		Name:        p.Name,
		Label:       p.Label.Get(lang),
		Description: p.Description.Get(lang),
		IsDefault:   p.IsDefault,
		Schema:      p.Schema,
	}
}

// LocalizedErrorCodeDef represents an error code with localized text
type LocalizedErrorCodeDef struct {
	Code        string        `json:"code"`
	Name        LocalizedText `json:"name"`
	Description LocalizedText `json:"description"`
	Retryable   bool          `json:"retryable"`
}

// ToErrorCodeDef converts to ErrorCodeDef for the specified language
func (e LocalizedErrorCodeDef) ToErrorCodeDef(lang string) ErrorCodeDef {
	return ErrorCodeDef{
		Code:        e.Code,
		Name:        e.Name.Get(lang),
		Description: e.Description.Get(lang),
		Retryable:   e.Retryable,
	}
}

// LocalizedConfigSchema holds config schemas for each language
type LocalizedConfigSchema struct {
	EN json.RawMessage `json:"en"`
	JA json.RawMessage `json:"ja"`
}

// Get returns the config schema for the specified language
func (s LocalizedConfigSchema) Get(lang string) json.RawMessage {
	switch lang {
	case "ja":
		if len(s.JA) > 0 && string(s.JA) != "null" {
			return s.JA
		}
		return s.EN
	case "en":
		if len(s.EN) > 0 && string(s.EN) != "null" {
			return s.EN
		}
		return s.JA
	default:
		if len(s.JA) > 0 && string(s.JA) != "null" {
			return s.JA
		}
		return s.EN
	}
}

// IsEmpty returns true if both schemas are empty
func (s LocalizedConfigSchema) IsEmpty() bool {
	return (len(s.EN) == 0 || string(s.EN) == "null" || string(s.EN) == "{}") &&
		(len(s.JA) == 0 || string(s.JA) == "null" || string(s.JA) == "{}}")
}
