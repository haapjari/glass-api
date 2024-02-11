package srv

import (
	"github.com/gin-gonic/gin"
	"github.com/haapjari/glass-api/api"
	"github.com/haapjari/glass-api/internal/pkg/svc"
	"github.com/haapjari/glass-api/internal/pkg/utils"
)

type DateResponse struct {
	Date string `json:"date,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error,omitempty"`
}

// TODO
func (s *Server) GetApiV1RepositoriesSearch(ctx *gin.Context, params api.GetApiV1RepositoriesSearchParams) {
	panic("TODO")
}

// GetApiV1RepositoriesSearchFirstCreationDate godoc
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

	ctx.IndentedJSON(status, &DateResponse{Date: firstCreationDate})
}

// TODO
func (s *Server) GetApiV1RepositoriesSearchLastCreationDate(ctx *gin.Context, params api.GetApiV1RepositoriesSearchLastCreationDateParams) {
	panic("TODO")
}
