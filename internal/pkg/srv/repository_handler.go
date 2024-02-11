package srv

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/haapjari/glass-api/api"
	"github.com/haapjari/glass-api/internal/pkg/svc"
	"github.com/haapjari/glass-api/internal/pkg/utils"
)

type ErrorResponse struct {
	Error string `json:"error,omitempty"`
}

// GetApiV1RepositoriesSearch godoc
// Handler for /api/v1/repositories/search.
func (s *Server) GetApiV1RepositoriesSearch(ctx *gin.Context, params api.GetApiV1RepositoriesSearchParams) {
	var (
		stars             = params.Stars
		language          = params.Language
		lastCreationDate  = params.LastCreationDate
		firstCreationDate = params.FirstCreationDate
		order             = fmt.Sprintf("%v", *params.Order)
		token, err        = utils.ParseBearer(ctx.GetHeader("Authorization"))
	)

	if err != nil {
		s.log.Warnf("request provided malformed or empty authorization header: %s", err.Error())
	}

	service, err := svc.NewRepositorySearchService(s.log, s.config, token)
	if err != nil {
		s.log.Errorf("error, while creating the repository search service: %s", err.Error())
		ctx.IndentedJSON(500, &ErrorResponse{Error: err.Error()})
		return
	}

	repositories, count, status, err := service.Search(language, stars, firstCreationDate, lastCreationDate, order)
	if err != nil {
		s.log.Errorf("error, while querying for first creation date: %s", err.Error())
		ctx.IndentedJSON(status, &ErrorResponse{Error: err.Error()})
		return
	}

	res := &api.GetApiV1RepositoriesSearchResponse{
		JSON200: &struct {
			Items      []api.Repository `json:"items"`
			TotalCount int              `json:"total_count"`
		}{
			TotalCount: count,
			Items:      repositories,
		},
	}

	ctx.IndentedJSON(status, res.JSON200)
}

// GetApiV1RepositoriesSearchFirstCreationDate godoc
// Handler for /api/v1/repositories/search/firstCreationDate.
func (s *Server) GetApiV1RepositoriesSearchFirstCreationDate(ctx *gin.Context,
	params api.GetApiV1RepositoriesSearchFirstCreationDateParams) {
	var (
		stars      = params.Stars
		language   = params.Language
		token, err = utils.ParseBearer(ctx.GetHeader("Authorization"))
	)

	if err != nil {
		s.log.Warnf("request provided malformed or empty authorization header: %s", err.Error())
	}

	service, err := svc.NewRepositorySearchService(s.log, s.config, token)
	if err != nil {
		s.log.Errorf("error, while creating the repository search service: %s", err.Error())
		ctx.IndentedJSON(500, &ErrorResponse{Error: err.Error()})
		return
	}

	firstCreationDate, status, err := service.FirstCreationDate(language, stars)
	if err != nil {
		s.log.Errorf("error, while querying for first creation date: %s", err.Error())
		ctx.IndentedJSON(status, &ErrorResponse{Error: err.Error()})
		return
	}

	res := &api.GetApiV1RepositoriesSearchFirstCreationDateResponse{
		JSON200: &struct {
			FirstCreationDate *string `json:"firstCreationDate,omitempty"`
		}{
			FirstCreationDate: &firstCreationDate,
		},
	}

	ctx.IndentedJSON(status, res.JSON200)
}

// GetApiV1RepositoriesSearchLastCreationDate godoc
// Handler for /api/v1/repositories/search/lastCreationDate.
func (s *Server) GetApiV1RepositoriesSearchLastCreationDate(ctx *gin.Context, params api.GetApiV1RepositoriesSearchLastCreationDateParams) {
	var (
		stars      = params.Stars
		language   = params.Language
		token, err = utils.ParseBearer(ctx.GetHeader("Authorization"))
	)

	if err != nil {
		s.log.Warnf("request provided malformed or empty authorization header: %s", err.Error())
	}

	service, err := svc.NewRepositorySearchService(s.log, s.config, token)
	if err != nil {
		s.log.Errorf("error, while creating the repository search service: %s", err.Error())
		ctx.IndentedJSON(500, &ErrorResponse{Error: err.Error()})
		return
	}

	lastCreationDate, status, err := service.LastCreationDate(language, stars)
	if err != nil {
		s.log.Errorf("error, while querying for first creation date: %s", err.Error())
		ctx.IndentedJSON(status, &ErrorResponse{Error: err.Error()})
		return
	}

	res := &api.GetApiV1RepositoriesSearchLastCreationDateResponse{
		JSON200: &struct {
			LastCreationDate *string `json:"lastCreationDate,omitempty"`
		}{
			LastCreationDate: &lastCreationDate,
		},
	}

	ctx.IndentedJSON(status, res.JSON200)
}
