package httpadapter

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/code-corhuila/drp-availability-api/internal/app"
	"github.com/code-corhuila/drp-availability-api/internal/domain"
)

var (
	canonicalUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	instantZ      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
)

type spaceDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Capacity  int    `json:"capacity"`
	Available bool   `json:"available"`
}

type listEnvelope struct {
	Data []spaceDTO `json:"data"`
	Meta metaDTO    `json:"meta"`
}

type metaDTO struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type availabilityDTO struct {
	SpaceID   string `json:"spaceId"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

func toSpaceDTO(s app.SpaceSnapshot) spaceDTO {
	return spaceDTO{ID: s.ID, Name: s.Name, Kind: s.Kind, Capacity: s.Capacity, Available: s.Active}
}

func ListAvailable(search app.Search) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		period, berr := parsePeriod(r)
		if berr != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", berr.msg, berr.details)
			return
		}
		page, limit, berr := parsePageLimit(r)
		if berr != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", berr.msg, berr.details)
			return
		}
		filter := app.ListFilter{Period: period, Page: page, Limit: limit}
		if raw := strings.TrimSpace(r.URL.Query().Get("kind")); raw != "" {
			switch raw {
			case "WORKSTATION", "MEETING_ROOM", "PRIVATE_OFFICE", "TRAINING_ROOM", "AUDITORIUM":
				filter.Kind = &raw
			default:
				writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "El campo kind no es válido", []map[string]string{{"field": "kind", "message": "Debe ser un tipo de espacio conocido"}})
				return
			}
		}
		if raw := strings.TrimSpace(r.URL.Query().Get("minCapacity")); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 {
				writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "El campo minCapacity no es válido", []map[string]string{{"field": "minCapacity", "message": "Debe ser un entero mayor o igual a 1"}})
				return
			}
			filter.MinCapacity = &n
		}
		got, err := search.ListAvailable(r.Context(), filter)
		if errors.Is(err, app.ErrValidation) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Parámetros de paginación inválidos", nil)
			return
		}
		if err != nil {
			writeErr(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Error interno", nil)
			return
		}
		data := make([]spaceDTO, 0, len(got.Items))
		for _, s := range got.Items {
			data = append(data, toSpaceDTO(s))
		}
		writeJSON(w, http.StatusOK, listEnvelope{
			Data: data,
			Meta: metaDTO{Page: got.Page, Limit: got.Limit, Total: got.Total, TotalPages: got.TotalPages},
		})
	}
}

func GetAvailability(search app.Search) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("spaceId")
		if !canonicalUUID.MatchString(id) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "El identificador no es un UUID canónico", []map[string]string{{"field": "spaceId", "message": "Debe ser un UUID en minúsculas con guiones"}})
			return
		}
		period, berr := parsePeriod(r)
		if berr != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", berr.msg, berr.details)
			return
		}
		got, err := search.One(r.Context(), id, period)
		if errors.Is(err, app.ErrNotFound) {
			writeErr(w, r, http.StatusNotFound, "NOT_FOUND", "Espacio no encontrado", nil)
			return
		}
		if err != nil {
			writeErr(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Error interno", nil)
			return
		}
		writeJSON(w, http.StatusOK, availabilityDTO{SpaceID: got.SpaceID, Available: got.Available, Reason: got.Reason})
	}
}

type bindError struct {
	msg     string
	details []map[string]string
}

func (e bindError) Error() string { return e.msg }

func parsePeriod(r *http.Request) (domain.DateTimeRange, *bindError) {
	start, err := parseInstant(r.URL.Query().Get("startAt"), "startAt")
	if err != nil {
		return domain.DateTimeRange{}, err
	}
	end, err := parseInstant(r.URL.Query().Get("endAt"), "endAt")
	if err != nil {
		return domain.DateTimeRange{}, err
	}
	if !end.After(start) {
		return domain.DateTimeRange{}, &bindError{
			msg:     "El campo endAt debe ser posterior a startAt",
			details: []map[string]string{{"field": "endAt", "message": "Debe ser posterior a startAt"}},
		}
	}
	p, perr := domain.NewDateTimeRange(start, end)
	if perr != nil {
		return domain.DateTimeRange{}, &bindError{
			msg:     "El campo endAt debe ser posterior a startAt",
			details: []map[string]string{{"field": "endAt", "message": "Debe ser posterior a startAt"}},
		}
	}
	return p, nil
}

func parseInstant(raw, field string) (time.Time, *bindError) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, &bindError{
			msg:     "El campo " + field + " es requerido",
			details: []map[string]string{{"field": field, "message": "Es requerido"}},
		}
	}
	if !instantZ.MatchString(raw) {
		return time.Time{}, &bindError{
			msg:     "El campo " + field + " debe ser RFC3339 en UTC con Z",
			details: []map[string]string{{"field": field, "message": "Debe ser RFC3339 en UTC terminado en Z"}},
		}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, &bindError{
			msg:     "El campo " + field + " debe ser RFC3339 en UTC con Z",
			details: []map[string]string{{"field": field, "message": "Debe ser RFC3339 en UTC terminado en Z"}},
		}
	}
	return t.UTC(), nil
}

func parsePageLimit(r *http.Request) (int, int, *bindError) {
	page := 1
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, &bindError{msg: "El parámetro page debe ser un entero", details: []map[string]string{{"field": "page", "message": "Debe ser un entero"}}}
		}
		if n < 1 {
			return 0, 0, &bindError{msg: "El parámetro page debe ser mayor o igual a 1", details: []map[string]string{{"field": "page", "message": "Debe ser mayor o igual a 1"}}}
		}
		page = n
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, &bindError{msg: "El parámetro limit debe ser un entero", details: []map[string]string{{"field": "limit", "message": "Debe ser un entero"}}}
		}
		if n < 1 || n > 100 {
			return 0, 0, &bindError{msg: "El parámetro limit debe estar entre 1 y 100", details: []map[string]string{{"field": "limit", "message": "Debe estar entre 1 y 100"}}}
		}
		limit = n
	}
	return page, limit, nil
}
