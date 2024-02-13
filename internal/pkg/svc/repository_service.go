package svc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-github/v59/github"
	"github.com/haapjari/glass-api/api"
	"github.com/haapjari/glass-api/internal/pkg/cfg"
	"github.com/haapjari/glass-api/internal/pkg/logger"
	"github.com/haapjari/glass-api/internal/pkg/utils"
	"io"
	"net/http"
	"time"
)

type RepositorySearchService struct {
	log                       logger.Logger
	gitHubPersonalAccessToken string
	gitHubQueryInterval       time.Duration
	httpClient                *http.Client
	gitHubClient              *github.Client
}

type Count struct {
	TotalCount int `json:"total_count"`
}

type RepositoryResponse struct {
	TotalCount int              `json:"total_count"`
	Items      []api.Repository `json:"items"`
}

func NewRepositorySearchService(logger logger.Logger, config *cfg.Config, token string) (*RepositorySearchService, error) {
	interval, err := time.ParseDuration(config.GitHubQueryInterval)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: time.Duration(30) * time.Second,
	}

	return &RepositorySearchService{
		log:                       logger,
		gitHubPersonalAccessToken: token,
		gitHubQueryInterval:       interval,
		httpClient:                httpClient,
		gitHubClient:              github.NewClient(httpClient).WithAuthToken(token),
	}, nil
}

func (rss *RepositorySearchService) Populate() {
	// TODO
}

// Search is an abstraction of GitHub Repository Search API.
// Returns slice of repositories, count, status and optionally an error.
// TODO
func (rss *RepositorySearchService) Search(language string, stars string, firstCreationDate string,
	lastCreationDate string, order string) ([]api.Repository, int, int, error) {
	if language == "" || stars == "" {
		return nil, 0, 400, errors.New("language or stars field is empty")
	}

	var (
		queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:%s..%s&order=%s",
			language, stars, firstCreationDate, lastCreationDate, order)
		endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
	)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, 500, err
	}

	rss.log.Debugf("Request: %v", endpoint)

	if rss.gitHubPersonalAccessToken != "" {
		req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
	}

	resp, err := rss.httpClient.Do(req)
	if err != nil {
		return nil, 0, 500, err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			rss.log.Warnf("unable to close response body: %s", err.Error())
		}
	}()

	rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

	if resp.StatusCode == 200 {
		repositoryResponse := &RepositoryResponse{}

		body, e := io.ReadAll(resp.Body)
		if e != nil {
			return nil, 0, 500, e
		}

		if err = json.Unmarshal(body, &repositoryResponse); err != nil {
			return nil, 0, 500, err
		}

		// Populate Contributor Count.
		// TODO: Create Populator Goroutine, and
		for i := range len(repositoryResponse.Items) {
			owner, name, loopErr := utils.ParseGitHubFullName(repositoryResponse.Items[i].FullName)
			if loopErr != nil {
				return nil, 0, 500, loopErr
			}

			rss.log.Debugf("[Owner: %v] [Name: %v] populating repository ", owner, name)

			contributorCount, loopErr := rss.GetContributorCount(owner, name)
			if loopErr != nil {
				return nil, 0, 500, loopErr
			}

			repositoryResponse.Items[i].ContributorsCount = &contributorCount
		}

		// TODO: Query for These.
		// latest_release
		// total_releases_count
		// contributors_count
		// open_pulls_count
		// closed_pulls_count
		// subscribers_count
		// commits_count
		// events_count
		// watchers

		// library_loc
		// self_written_loc

		return repositoryResponse.Items, repositoryResponse.TotalCount, 200, nil
	}

	return nil, 0, 500, nil
}

// GetContributorCount godoc
func (rss *RepositorySearchService) GetContributorCount(owner string, name string) (int, error) {
	opt := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
		Anon: "true",
	}

	var all []*github.Contributor

	for {
		contributors, resp, err := rss.gitHubClient.Repositories.ListContributors(context.Background(),
			owner, name, opt)
		if err != nil {
			return -1, err
		}

		all = append(all, contributors...)
		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return len(all), nil
}

// LastCreationDate queries GitHub Search API and returns the last creation date matching the query parameters.
// Returns date as string, API Status Code and error.
func (rss *RepositorySearchService) LastCreationDate(language string, stars string) (string, int, error) {
	if language == "" || stars == "" {
		return "", 400, errors.New("language or stars field is empty")
	}

	var (
		date  = time.Now()
		year  = date.Year()
		month = int(date.Month())
		day   = date.Day()
	)

	startDate, e := time.Parse(time.DateOnly, fmt.Sprintf("%d-%02d-%02d", year, month, day))
	if e != nil {
		return "", 500, e
	}

	totalCount := -1

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:>%s&order=asc",
					language, stars, startDate.Format(time.DateOnly))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return "", 500, err
			}

			rss.log.Debugf("Request: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return "", 500, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return "", 500, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return "", 500, err
				}

				totalCount = count.TotalCount

				if totalCount != 0 {
					return startDate.Format(time.DateOnly), 200, nil
				}

				startDate = startDate.AddDate(0, 0, -1)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
		}
	}
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

			rss.log.Debugf("Request: %v", endpoint)

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

				if totalCount != 0 {
					return startDate.Year() - 1, nil
				}

				startDate = startDate.AddDate(1, 0, 0)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
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

			rss.log.Debugf("Request: %v", endpoint)

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

				if totalCount != 0 {
					return int(startDate.Month() - 1), nil
				}

				startDate = startDate.AddDate(0, 1, 0)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
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

			rss.log.Debugf("Request: %v", endpoint)

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

				if totalCount != 0 {
					return startDate.AddDate(0, 0, -7).Day(), nil
				}

				startDate = startDate.AddDate(0, 0, 7)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
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

			rss.log.Debugf("Request: %v", endpoint)

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

				if totalCount != 0 {
					return startDate.AddDate(0, 0, -1).Day(), nil
				}

				startDate = startDate.AddDate(0, 0, 1)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
		}
	}
}
