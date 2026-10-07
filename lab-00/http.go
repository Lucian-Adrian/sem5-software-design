package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type API struct {
	groups   *GroupService
	profiles *ProfileService
}

type joinInput struct {
	ProfileID string `json:"profileId"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func NewHTTPHandler(groups *GroupService, profiles *ProfileService) http.Handler {
	api := &API{groups: groups, profiles: profiles}

	// Route HTTP requests to handlers; handlers leave business rules to services.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /groups", api.listGroups)
	mux.HandleFunc("POST /groups", api.createGroup)
	mux.HandleFunc("GET /groups/{id}", api.getGroup)
	mux.HandleFunc("POST /groups/{id}/join", api.joinGroup)
	mux.HandleFunc("POST /profiles", api.createProfile)
	return mux
}

func (api *API) listGroups(w http.ResponseWriter, r *http.Request) {
	// Select the listing rule from the optional budget query.
	var filter GroupFilter = AllGroups{}
	if budgetValues, found := r.URL.Query()["budget"]; found {
		if len(budgetValues) != 1 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "budget must appear once")
			return
		}
		budget, err := strconv.Atoi(budgetValues[0])
		if err != nil || budget < 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "budget must be a non-negative whole number")
			return
		}
		filter = AffordableGroups{MonthlyBudget: budget}
	}

	groups, err := api.groups.ListGroups(filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (api *API) createGroup(w http.ResponseWriter, r *http.Request) {
	var input GroupInput
	if err := decodeJSONBody(w, r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	group, err := api.groups.CreateGroup(input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (api *API) getGroup(w http.ResponseWriter, r *http.Request) {
	group, err := api.groups.GetGroup(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (api *API) joinGroup(w http.ResponseWriter, r *http.Request) {
	var input joinInput
	if err := decodeJSONBody(w, r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if input.ProfileID == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "profileId is required")
		return
	}

	group, err := api.groups.JoinGroup(r.PathValue("id"), input.ProfileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (api *API) createProfile(w http.ResponseWriter, r *http.Request) {
	var input ProfileInput
	if err := decodeJSONBody(w, r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	profile, err := api.profiles.CreateProfile(input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, profile)
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, destination any) error {
	// Bound the input size and reject fields outside the request model.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("request body must contain valid JSON: %w", err)
	}

	// A second JSON value would make one request ambiguous.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain one JSON value")
		}
		return fmt.Errorf("request body must contain valid JSON: %w", err)
	}
	return nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	// Translate domain outcomes to HTTP only at the transport boundary.
	switch {
	case errors.Is(err, ErrGroupNotFound), errors.Is(err, ErrProfileNotFound):
		writeAPIError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrGroupFull):
		writeAPIError(w, http.StatusConflict, "group_full", err.Error())
	case errors.Is(err, ErrBudgetTooLow):
		writeAPIError(w, http.StatusUnprocessableEntity, "budget_too_low", err.Error())
	case errors.Is(err, ErrInvalidInput):
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "the request could not be completed")
	}
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
