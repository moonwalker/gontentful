package content

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const (
	JsonSchemaName = "_schema.json"
	DefaultLocale  = "en"
)

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
	Schema       *Schema       `json:"schema,omitempty"`
}

type Fields []*Field

func (f *Fields) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		json.Unmarshal(v, f)
		return nil
	case string:
		json.Unmarshal([]byte(v), f)
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}
func (f Fields) Value() (driver.Value, error) {
	b, err := json.Marshal(f)
	return string(b), err
}

type Schema struct {
	ID           string     `json:"id,omitempty"`
	Name         string     `json:"name,omitempty"`
	DisplayField string     `json:"displayField,omitempty"`
	Description  string     `json:"description,omitempty"`
	Fields       Fields     `json:"fields,omitempty"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
	CreatedBy    string     `json:"createdBy,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    string     `json:"updatedBy,omitempty"`
	Version      int        `json:"version,omitempty"`
}

type ContentData struct {
	ID        string                 `json:"id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	CreatedAt string                 `json:"createdAt,omitempty"`
	CreatedBy string                 `json:"createdBy,omitempty"`
	UpdatedAt string                 `json:"updatedAt,omitempty"`
	UpdatedBy string                 `json:"updatedBy,omitempty"`
	Version   int                    `json:"version,omitempty"`
}

type MergedContentData struct {
	ID        string                            `json:"id,omitempty"`
	Name      string                            `json:"name,omitempty"`
	Type      string                            `json:"type,omitempty"`
	Fields    map[string]map[string]interface{} `json:"fields,omitempty"`
	CreatedAt *time.Time                        `json:"createdAt,omitempty"`
	CreatedBy string                            `json:"createdBy,omitempty"`
	UpdatedAt *time.Time                        `json:"updatedAt,omitempty"`
	UpdatedBy string                            `json:"updatedBy,omitempty"`
	Version   int                               `json:"version,omitempty"`
}
