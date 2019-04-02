package gontentful

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

type SyncCallback func(*SyncResponse) error

func (s *SpacesService) Sync(token string) (*SyncResult, error) {
	var err error
	res := &SyncResult{}

	res.Token, err = s.SyncPaged(token, func(sr *SyncResponse) error {
		for _, item := range sr.Items {
			res.Items = append(res.Items, item)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = resolveResponse(res.Items)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *SpacesService) SyncPaged(token string, callback SyncCallback) (string, error) {
	query := url.Values{}
	if len(token) == 0 {
		query.Set("initial", "true")
	} else {
		query.Set("sync_token", token)
	}

	res, err := s.getSyncPage(query)
	if err != nil {
		return "", err
	}

	err = callback(res)
	if err != nil {
		return "", err
	}

	if len(res.NextPageURL) > 0 {
		t, err := getSyncToken(res.NextPageURL)
		if err != nil {
			return "", err
		}
		return s.SyncPaged(t, callback)
	}

	return getSyncToken(res.NextSyncURL)
}

func (s *SpacesService) getSyncPage(query url.Values) (*SyncResponse, error) {
	path := fmt.Sprintf(pathSync, s.client.Options.SpaceID)
	body, err := s.client.get(path, query)
	if err != nil {
		return nil, err
	}

	res := &SyncResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func getSyncToken(pageUrl string) (string, error) {
	npu, _ := url.Parse(pageUrl)
	m, _ := url.ParseQuery(npu.RawQuery)

	syncToken := m.Get("sync_token")
	if syncToken == "" {
		return "", fmt.Errorf("missing sync token from response: %s", pageUrl)
	}
	return syncToken, nil
}

// cf js sdk: https://github.com/contentful/contentful-resolve-response
func resolveResponse(items []*Entry) error {
	if len(items) == 0 {
		return nil
	}

	// get locales from db. will need to sync space into it's own table
	locales := []string{"en", "sv", "en-SE"}

	// map by id for easier access
	entryMap := make(map[string]*Entry)
	assetMap := make(map[string]*Entry)

	for _, item := range items {
		id := item.Sys.ID
		switch item.Sys.Type {
		case "Entry":
			entryMap[id] = item
			break
		case "Asset":
			assetMap[id] = item
			break
		case "DeletedEntry":
			// remove by id (don't know the type)
			// remove where linked
			break
		case "DeletedAsset":
			// remove by id from _assets
			// remove where linked
			break
		}
	}

	for _, entry := range entryMap {
		for _, field := range entry.Fields {
			f, ok := field.(map[string]interface{})
			if ok {
				for _, l := range locales {
					ft := reflect.TypeOf(f[l])
					if ft != nil {
						fieldType := ft.String()
						if strings.HasPrefix(fieldType, "[]") {
							arr, ok := f[l].([]interface{})
							if ok {
								for _, af := range arr {
									resolveEntry(af, entryMap, assetMap)
								}
							} else if fieldType != "int" && fieldType != "float64" && fieldType != "bool" && fieldType == "string" {
								resolveEntry(f[l], entryMap, assetMap)
							}
						}
					}
				}
			}
		}
	}

	// for id, ct := range includedEntry {
	// 	if itemMap[id] != nil {
	// 		changedIncludes = append(changedIncludes, itemMap[id])
	// 	} else {
	// 		// get item from database
	// 		fmt.Println("get item from database", ct, id)

	// 	}
	// }

	// for id := range includedAsset {
	// 	if itemMap[id] != nil {
	// 		changedIncludes = append(changedIncludes, itemMap[id])
	// 	} else {
	// 		// get asset from database
	// 		fmt.Println("get asset from database", id)
	// 	}
	// }

	return nil
}

func resolveEntry(entry interface{}, entryMap map[string]*Entry, assetMap map[string]*Entry) interface{} {
	e, ok := entry.(*Entry) //.(map[string]interface{})
	if ok {
		if e.Sys.Type == "Link" {
			if e.Sys.LinkType == "Entry" {
				if entryMap[e.Sys.ID] == nil {
					// get from db
				} else {
					return entryMap[e.Sys.ID].Fields
				}
			} else if e.Sys.LinkType == "Asset" {
				if assetMap[e.Sys.ID] == nil {
					// get from db
				} else {
					return assetMap[e.Sys.ID].Fields
				}
			}
		}
	}
	return entry
}
