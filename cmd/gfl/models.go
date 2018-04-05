package main

type Resp struct {
	Total int           `json:"total"`
	Limit int           `json:"limit"`
	Skip  int           `json:"skip"`
	Items []ContentType `json:"items"`
}

type ContentType struct {
	Sys          *Sys     `json:"sys"`
	Name         string   `json:"name,omitempty"`
	Description  string   `json:"description,omitempty"`
	Fields       []*Field `json:"fields,omitempty"`
	DisplayField string   `json:"displayField,omitempty"`
}

type Field struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	LinkType    string              `json:"linkType,omitempty"`
	Items       *FieldTypeArrayItem `json:"items,omitempty"`
	Required    bool                `json:"required,omitempty"`
	Localized   bool                `json:"localized,omitempty"`
	Disabled    bool                `json:"disabled,omitempty"`
	Omitted     bool                `json:"omitted,omitempty"`
	Validations []FieldValidation   `json:"validations,omitempty"`
}

type FieldTypeArrayItem struct {
	Type        string                `json:"type,omitempty"`
	Validations []FieldValidation `json:"validations,omitempty"`
	LinkType    string                `json:"linkType,omitempty"`
}

type FieldValidation struct {
	LinkContentType []string `json:"linkContentType"`
	LinkMimetypeGroup []string `json:"linkMimetypeGroup"`
}

type Sys struct {
	ID               string       `json:"id,omitempty"`
	Type             string       `json:"type,omitempty"`
	LinkType         string       `json:"linkType,omitempty"`
	CreatedAt        string       `json:"createdAt,omitempty"`
	UpdatedAt        string       `json:"updatedAt,omitempty"`
	UpdatedBy        *Sys         `json:"updatedBy,omitempty"`
	Version          int          `json:"version,omitempty"`
	Revision         int          `json:"revision,omitempty"`
	ContentType      *ContentType `json:"contentType,omitempty"`
	FirstPublishedAt string       `json:"firstPublishedAt,omitempty"`
	PublishedCounter int          `json:"publishedCounter,omitempty"`
	PublishedAt      string       `json:"publishedAt,omitempty"`
	PublishedBy      *Sys         `json:"publishedBy,omitempty"`
	PublishedVersion int          `json:"publishedVersion,omitempty"`
}
