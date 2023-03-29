package content

import (
	"time"
)

const JsonSchemaName = "_schema.json"

type Asset struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	FileName    string     `json:"file_name,omitempty"`
	ContentType string     `json:"content_type,omitempty"`
	URL         string     `json:"url,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	CreatedBy   string     `json:"createdBy,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy   string     `json:"updatedBy,omitempty"`
	Version     int        `json:"version,omitempty"`
}

type Validation struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type Field struct {
	ID           string        `json:"id,omitempty"`
	Label        string        `json:"label"`
	Type         string        `json:"type"`
	Reference    bool          `json:"reference,omitempty"`
	List         bool          `json:"list,omitempty"`
	Localized    bool          `json:"localized,omitempty"`
	Disabled     bool          `json:"disabled,omitempty"`
	DefaultValue interface{}   `json:"defaultValue,omitempty"`
	Validations  []*Validation `json:"validations,omitempty"`
}

type Schema struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Fields      []*Field   `json:"fields,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	CreatedBy   string     `json:"createdBy,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy   string     `json:"updatedBy,omitempty"`
	Version     int        `json:"version,omitempty"`
}

type ContentData struct {
	ID        string                 `json:"id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	CreatedAt *time.Time             `json:"createdAt,omitempty"`
	CreatedBy string                 `json:"createdBy,omitempty"`
	UpdatedAt *time.Time             `json:"updatedAt,omitempty"`
	UpdatedBy string                 `json:"updatedBy,omitempty"`
	Version   int                    `json:"version,omitempty"`
}
