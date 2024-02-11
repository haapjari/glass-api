package svc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/haapjari/glass-api/internal/pkg/cfg"
	"github.com/haapjari/glass-api/internal/pkg/logger"
	"io"
	"net/http"
	"time"
)

type RepositorySearchService struct {
	log                       logger.Logger
	gitHubPersonalAccessToken string
	gitHubQueryInterval       time.Duration
	httpClient                *http.Client
}

type Count struct {
	TotalCount int `json:"total_count"`
}

func NewRepositorySearchService(logger logger.Logger, config *cfg.Config, token string) (*RepositorySearchService, error) {
	interval, err := time.ParseDuration(config.GitHubQueryInterval)
	if err != nil {
		return nil, err
	}

	return &RepositorySearchService{
		log:                       logger,
		gitHubPersonalAccessToken: token,
		gitHubQueryInterval:       interval,
		httpClient: &http.Client{
			Timeout: time.Duration(30) * time.Second,
		},
	}, nil
}

// FirstCreationDate queries GitHub Search API and returns the first creation date matching the query parameters.
// Returns date as string, API Status Code and error.
func (rss *RepositorySearchService) FirstCreationDate(language string, stars string) (string, int, error) {
	if language == "" || stars == "" {
		return "", 400, errors.New("language or stars field is empty")
	}

	year, err := rss.findFirstYear(language, stars)
	if err != nil {
		return "", 500, err
	}

	month, err := rss.findFirstMonth(language, stars, year)
	if err != nil {
		return "", 500, err
	}

	weekDay, err := rss.findFirstWeek(language, stars, year, month)
	if err != nil {
		return "", 500, err
	}

	day, err := rss.findFirstDay(language, stars, year, month, weekDay)
	if err != nil {
		return "", 500, err
	}

	return fmt.Sprintf("%d-%02d-%02d", year, month, day), 200, nil
}

func (rss *RepositorySearchService) findFirstYear(language string, stars string) (int, error) {
	startDate, e := time.Parse("2006-01-02", "2007-01-01")
	if e != nil {
		return -1, e
	}

	totalCount := -1

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:<%s&order=asc",
					language, stars, startDate.Format("2006-01-02"))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Request to the GitHub API Endpoint: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return -1, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return -1, err
				}

				totalCount = count.TotalCount
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}

			rss.log.Debugf("Date: %v, totalCount: %v", startDate.Format("2006-01-02"), totalCount)

			if totalCount != 0 {
				return startDate.Year() - 1, nil
			}

			startDate = startDate.AddDate(1, 0, 0)
		}
	}
}

func (rss *RepositorySearchService) findFirstMonth(language string, stars string, year int) (int, error) {
	startDate, e := time.Parse("2006-01-02", fmt.Sprintf("%v-01-01", year))
	if e != nil {
		return -1, e
	}

	totalCount := -1

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:<%s&order=asc",
					language, stars, startDate.Format("2006-01-02"))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Request to the GitHub API Endpoint: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return -1, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return -1, err
				}

				totalCount = count.TotalCount
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}

			rss.log.Debugf("Date: %v, totalCount: %v", startDate.Format("2006-01-02"), totalCount)

			if totalCount != 0 {
				return int(startDate.Month() - 1), nil
			}

			startDate = startDate.AddDate(0, 1, 0)
		}
	}
}

func (rss *RepositorySearchService) findFirstWeek(language string, stars string, year int, month int) (int, error) {
	startDate, e := time.Parse("2006-01-02", fmt.Sprintf("%d-%02d-%02d", year, month, 1))
	if e != nil {
		return -1, e
	}

	totalCount := -1

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:<%s&order=asc",
					language, stars, startDate.Format("2006-01-02"))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Request to the GitHub API Endpoint: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return -1, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return -1, err
				}

				totalCount = count.TotalCount
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}

			rss.log.Debugf("Date: %v, totalCount: %v", startDate.Format("2006-01-02"), totalCount)

			if totalCount != 0 {
				return startDate.AddDate(0, 0, -7).Day(), nil
			}

			startDate = startDate.AddDate(0, 0, 7)
		}
	}
}

func (rss *RepositorySearchService) findFirstDay(language string, stars string, year int, month int, day int) (int, error) {
	startDate, e := time.Parse("2006-01-02", fmt.Sprintf("%d-%02d-%02d", year, month, day))
	if e != nil {
		return -1, e
	}

	totalCount := -1

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:<%s&order=asc",
					language, stars, startDate.Format("2006-01-02"))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Request to the GitHub API Endpoint: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return -1, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return -1, err
				}

				totalCount = count.TotalCount
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}

			rss.log.Debugf("Date: %v, totalCount: %v", startDate.Format("2006-01-02"), totalCount)

			if totalCount != 0 {
				return startDate.AddDate(0, 0, -1).Day(), nil
			}

			startDate = startDate.AddDate(0, 0, 1)
		}
	}
}
