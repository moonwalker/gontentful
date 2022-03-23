package gontentful

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	path := fmt.Sprintf(pathSync, s.client.Options.SpaceID, s.client.Options.EnvironmentID)
	// key := query.Get("sync_token")
	// if key == "" {
	// 	key = "initial"
	// } else {
	// 	key = string(key[len(key)-100:])
	// }
	// dat, err := ioutil.ReadFile("/tmp/sync/" + key)
	// if err == nil {
	// 	res := &SyncResponse{}
	// 	err = json.Unmarshal(dat, &res)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return res, nil
	// }
	body, err := s.client.get(path, query)
	if err != nil {
		return nil, err
	}
	res := &SyncResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	// ioutil.WriteFile("/tmp/sync/"+key, body, 0644)

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
