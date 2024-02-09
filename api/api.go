package api

import (
	"github.com/gorilla/mux"
	"github.com/temelpa/timetravel/service"
)

type API struct {
	records service.RecordService
}

func NewAPI(records service.RecordService) *API {
	return &API{records}
}

// generates all api routes
func (a *API) CreateRoutes(routes *mux.Router) {
	routes.Path("/records/{id}").HandlerFunc(a.GetRecords).Methods("GET")
	routes.Path("/records/{id}").HandlerFunc(a.PostRecords).Methods("POST")
}

func (a *API) CreateRoutes2(routes *mux.Router) {
	routes.Path("/records/{id}").HandlerFunc(a.GetRecords2).Methods("GET")
	routes.Path("/records/{id}/{versionId}").HandlerFunc(a.GetRecords2).Methods("GET")
	routes.Path("/recordVersions/{id}").HandlerFunc(a.GetVersionIdsForRecord).Methods("GET")
	routes.Path("/records/{id}").HandlerFunc(a.PostRecords2).Methods("POST")
}
