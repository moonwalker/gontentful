package gontentful

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type SyncService service

func (s *SyncService) Sync(token string, callback func(*SyncResponse) error) (string, error) {
	query := url.Values{}
	if token == "" {
		query.Set("initial", "true")
	} else {
		query.Set("sync_token", token)
	}

	res, err := s.sync(query)
	if err != nil {
		return "", err
	}
	err = callback(res)
	if err != nil {
		return "", err
	}

	if res.NextPageURL != "" {
		t, err := getSyncToken(res.NextPageURL)
		if err != nil {
			return "", err
		}
		return s.Sync(t, callback)
	}

	return getSyncToken(res.NextSyncURL)
}

func (s *SyncService) sync(query url.Values) (*SyncResponse, error) {
	path := fmt.Sprintf(pathSync, s.client.Options.SpaceID)
	body, err := s.client.get(path, query)
	if err != nil {
		return nil, err
	}
	res := &SyncResponse{}
	err = json.Unmarshal(body, res)
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
		return "", fmt.Errorf("Missing sync token from response: %s", pageUrl)
	}
	return syncToken, nil
}
