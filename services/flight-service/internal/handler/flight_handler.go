package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	sharederrors "github.com/flighttracker/pkg/errors"
	"github.com/flighttracker/pkg/response"
	"github.com/flighttracker/pkg/validator"

	"github.com/flighttracker/services/flight-service/internal/api"
	"github.com/flighttracker/services/flight-service/internal/model"
	"github.com/flighttracker/services/flight-service/internal/service"
)

const defaultRouteLimit = 10

// FlightHandler implements the three routes in
// docs/api-contracts/flight-service.yaml.
type FlightHandler struct {
	search *service.FlightSearchService
	ingest *service.IngestService
}

// searchAndIngest bundles both use-case services; kept as two fields on
// one handler (rather than two handlers) since both routes share the same
// request-parsing helpers below.
type SearchAndIngest struct {
	Search *service.FlightSearchService
	Ingest *service.IngestService
}

func NewFlightHandler(deps SearchAndIngest) *FlightHandler {
	return &FlightHandler{search: deps.Search, ingest: deps.Ingest}
}

// GetByFlightNumber handles GET /api/v1/flights/:flightNumber.
func (h *FlightHandler) GetByFlightNumber(c *gin.Context) {
	flightNumber := strings.ToUpper(c.Param("flightNumber"))
	if !validator.IsFlightNumber(flightNumber) {
		response.Error(c, sharederrors.BadRequest("INVALID_FLIGHT_NUMBER", "flightNumber is not a valid flight number"))
		return
	}

	date, err := parseDate(c.Query("date"))
	if err != nil {
		response.Error(c, sharederrors.BadRequest("INVALID_DATE", "date must be in YYYY-MM-DD format"))
		return
	}

	flight, err := h.search.GetByFlightNumber(c.Request.Context(), flightNumber, date)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toFlightResponseDTO(flight))
}

// SearchByRoute handles GET /api/v1/flights/route.
func (h *FlightHandler) SearchByRoute(c *gin.Context) {
	origin := strings.ToUpper(c.Query("origin"))
	destination := strings.ToUpper(c.Query("destination"))
	if !validator.IsIATA(origin) || !validator.IsIATA(destination) {
		response.Error(c, sharederrors.BadRequest("INVALID_IATA_CODE", "origin/destination must be valid 3-letter IATA codes"))
		return
	}

	date, err := parseDate(c.Query("date"))
	if err != nil {
		response.Error(c, sharederrors.BadRequest("INVALID_DATE", "date must be in YYYY-MM-DD format"))
		return
	}

	limit := defaultRouteLimit
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			response.Error(c, sharederrors.BadRequest("INVALID_LIMIT", "limit must be a non-negative integer"))
			return
		}
		limit = parsed
	}

	flights, err := h.search.SearchByRoute(c.Request.Context(), origin, destination, date, limit)
	if err != nil {
		response.Error(c, err)
		return
	}

	items := make([]api.FlightResponseDTO, 0, len(flights))
	for _, f := range flights {
		items = append(items, toFlightResponseDTO(f))
	}
	response.OK(c, api.RouteSearchResponseDTO{Items: items, Total: len(items)})
}

// Ingest handles POST /internal/v1/flights/ingest.
func (h *FlightHandler) Ingest(c *gin.Context) {
	var req api.IngestRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, sharederrors.BadRequest("INVALID_INGEST_PAYLOAD", err.Error()))
		return
	}

	snapshot := model.Flight{
		FlightNumber:       req.FlightNumber,
		AirlineIATA:        req.AirlineIATA,
		OriginIATA:         req.OriginIATA,
		DestinationIATA:    req.DestinationIATA,
		ScheduledDeparture: req.ScheduledDeparture,
		EstimatedDeparture: req.EstimatedDeparture,
		ActualDeparture:    req.ActualDeparture,
		ScheduledArrival:   req.ScheduledArrival,
		EstimatedArrival:   req.EstimatedArrival,
		ActualArrival:      req.ActualArrival,
		Gate:               req.Gate,
		Terminal:           req.Terminal,
		Status:             model.FlightStatus(req.Status),
		AircraftType:       req.AircraftType,
	}
	if req.AirlineName != "" {
		snapshot.AirlineName = &req.AirlineName
	}

	result, err := h.ingest.Ingest(c.Request.Context(), snapshot, req.Source, req.RawPayload)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, api.IngestResponseDTO{
		Flight:         toFlightResponseDTO(result.Flight),
		PreviousStatus: result.PreviousStatus,
		Changed:        result.Changed,
	})
}

func parseDate(raw string) (time.Time, error) {
	if raw == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Parse("2006-01-02", raw)
}

func toFlightResponseDTO(f model.Flight) api.FlightResponseDTO {
	return api.FlightResponseDTO{
		ID:                 f.ID.String(),
		FlightNumber:       f.FlightNumber,
		AirlineIATA:        f.AirlineIATA,
		AirlineName:        f.AirlineName,
		OriginIATA:         f.OriginIATA,
		DestinationIATA:    f.DestinationIATA,
		ScheduledDeparture: f.ScheduledDeparture,
		EstimatedDeparture: f.EstimatedDeparture,
		ActualDeparture:    f.ActualDeparture,
		ScheduledArrival:   f.ScheduledArrival,
		EstimatedArrival:   f.EstimatedArrival,
		ActualArrival:      f.ActualArrival,
		Gate:               f.Gate,
		Terminal:           f.Terminal,
		Status:             string(f.Status),
		AircraftType:       f.AircraftType,
		LastSyncedAt:       f.LastSyncedAt,
	}
}
