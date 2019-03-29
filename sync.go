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

	nextPageURL := res.NextPageURL
	nextSyncURL := res.NextSyncURL

	for nextPageURL != "" {
		t, err := getSyncToken(nextPageURL)
		if err != nil {
			return "", err
		}
		q := url.Values{}
		q.Set("sync_token", t)
		page, err := s.sync(q)
		if err != nil {
			return "", err
		}

		err = callback(page)
		if err != nil {
			return "", err
		}
		nextPageURL = page.NextPageURL
		nextSyncURL = page.NextSyncURL
	}

	nextSyncToken, err := getSyncToken(nextSyncURL)
	if err != nil {
		return "", err
	}

	return nextSyncToken, nil
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
