package gontentful

type Sys struct {
	ID               string       `json:"id,omitempty"`
	Type             string       `json:"type,omitempty"`
	LinkType         string       `json:"linkType,omitempty"`
	CreatedAt        string       `json:"createdAt,omitempty"`
	CreatedBy        *Entry       `json:"createdBy,omitempty"`
	UpdatedAt        string       `json:"updatedAt,omitempty"`
	UpdatedBy        *Entry       `json:"updatedBy,omitempty"`
	DeletedAt        string       `json:"deletedAt,omitempty"`
	DeletedBy        *Entry       `json:"deletedBy,omitempty"`
	Version          int          `json:"version,omitempty"`
	Revision         int          `json:"revision,omitempty"`
	ContentType      *ContentType `json:"contentType,omitempty"`
	FirstPublishedAt string       `json:"firstPublishedAt,omitempty"`
	PublishedCounter int          `json:"publishedCounter,omitempty"`
	PublishedAt      string       `json:"publishedAt,omitempty"`
	PublishedBy      *Entry       `json:"publishedBy,omitempty"`
	PublishedVersion int          `json:"publishedVersion,omitempty"`
	Space            *Space       `json:"space,omitempty"`
}

type Entries struct {
	Sys      *Sys     `json:"sys"`
	Total    int      `json:"total"`
	Skip     int      `json:"skip"`
	Limit    int      `json:"limit"`
	Items    []*Entry `json:"items"`
	Includes *Include `json:"includes,omitempty"`
}

type Include struct {
	Entry []*Entry `json:"entry,omitempty"`
	Asset []*Entry `json:"asset,omitempty"`
}

type Fields map[string]interface{}

type Entry struct {
	Sys    *Sys   `json:"sys"`
	Locale string `json:"locale,omitempty"`
	Fields Fields `json:"fields"` // fields are dynamic
}

type Space struct {
	Sys     *Sys      `json:"sys"`
	Name    string    `json:"name"`
	Locales []*Locale `json:"locales"`
}

type Spaces struct {
	Sys   *Sys     `json:"sys"`
	Total int      `json:"total"`
	Limit int      `json:"limit"`
	Skip  int      `json:"skip"`
	Items []*Space `json:"items"`
}

type Locales struct {
	Total int       `json:"total"`
	Limit int       `json:"limit"`
	Skip  int       `json:"skip"`
	Items []*Locale `json:"items"`
}

type Locale struct {
	Code         string   `json:"code"`
	Default      bool     `json:"default"`
	Name         string   `json:"name"`
	FallbackCode string   `json:"fallbackCode"`
	CFLocales    []string `json:"cfFallbackCode"`
}

type ContentType struct {
	Sys          *Sys                `json:"sys"`
	Name         string              `json:"name,omitempty"`
	Description  string              `json:"description,omitempty"`
	Fields       []*ContentTypeField `json:"fields,omitempty"`
	DisplayField string              `json:"displayField,omitempty"`
}

type ContentTypes struct {
	Total int            `json:"total"`
	Limit int            `json:"limit"`
	Skip  int            `json:"skip"`
	Items []*ContentType `json:"items"`
}

type ContentTypeField struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	LinkType    string              `json:"linkType,omitempty"`
	Items       *FieldTypeArrayItem `json:"items,omitempty"`
	Required    bool                `json:"required,omitempty"`
	Localized   bool                `json:"localized,omitempty"`
	Disabled    bool                `json:"disabled,omitempty"`
	Omitted     bool                `json:"omitted,omitempty"`
	Validations []*FieldValidation  `json:"validations,omitempty"`
}

type FieldTypeArrayItem struct {
	Type        string             `json:"type,omitempty"`
	Validations []*FieldValidation `json:"validations,omitempty"`
	LinkType    string             `json:"linkType,omitempty"`
}

type FieldValidation struct {
	LinkContentType   []string `json:"linkContentType"`
	LinkMimetypeGroup []string `json:"linkMimetypeGroup"`
	Unique            bool     `json:"unique"`
}

type CreateSpace struct {
	Name          string `json:"name"`
	DefaultLocale string `json:"defaultLocale"`
}

type SyncResponse struct {
	Sys         *Sys     `json:"sys"`
	Items       []*Entry `json:"items"`
	NextPageURL string   `json:"nextPageUrl"`
	NextSyncURL string   `json:"nextSyncUrl"`
}

type SyncResult struct {
	Items []*Entry
	Token string
}

type AssetFields struct {
	Title map[string]string
	File  map[string]*AssetFile
}

type AssetFile struct {
	URL         string `json:"url"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Details     struct {
		Size  int `json:"size"`
		Image struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"image"`
	} `json:"details"`
}

type PublishFields map[string]map[string]interface{}

type PublishedEntry struct {
	Sys    *Sys          `json:"sys"`
	Fields PublishFields `json:"fields"`
}

type PublishedEntries struct {
	Sys      *Sys              `json:"sys"`
	Total    int               `json:"total"`
	Skip     int               `json:"skip"`
	Limit    int               `json:"limit"`
	Items    []*PublishedEntry `json:"items"`
	Includes *Include          `json:"includes,omitempty"`
}
