package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type machineResponse struct {
	ID              uuid.UUID  `json:"id"`
	LicenseID       uuid.UUID  `json:"license_id"`
	Fingerprint     string     `json:"fingerprint"`
	Name            *string    `json:"name"`
	Hostname        *string    `json:"hostname"`
	DisplayName     string     `json:"display_name"`
	FirstSeenAt     string     `json:"first_seen_at"`
	LastSeenAt      string     `json:"last_seen_at"`
	LastSeenIP      *string    `json:"last_seen_ip"`
	ValidationCount int64      `json:"validation_count"`
	DeactivatedAt   *string    `json:"deactivated_at"`
	DeactivatedBy   *uuid.UUID `json:"deactivated_by"`
}

type updateMachineRequest struct {
	Name *string `json:"name"`
}

func machineToResponse(m *store.Machine) machineResponse {
	resp := machineResponse{
		ID:              m.ID,
		LicenseID:       m.LicenseID,
		Fingerprint:     m.Fingerprint,
		Name:            m.Name,
		Hostname:        m.Hostname,
		DisplayName:     store.MachineDisplayName(m),
		FirstSeenAt:     m.FirstSeenAt.Format(timeRFC3339),
		LastSeenAt:      m.LastSeenAt.Format(timeRFC3339),
		LastSeenIP:      m.LastSeenIP,
		ValidationCount: m.ValidationCount,
		DeactivatedBy:   m.DeactivatedBy,
	}
	if m.DeactivatedAt != nil {
		formatted := m.DeactivatedAt.Format(timeRFC3339)
		resp.DeactivatedAt = &formatted
	}
	return resp
}

func (s *Server) handleListLicenseMachines(w http.ResponseWriter, r *http.Request) {
	licenseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	lic, err := s.store.GetLicenseByID(r.Context(), licenseID)
	if err != nil {
		writeInternalError(w, r, err, "failed to load license")
		return
	}
	if lic == nil {
		writeError(w, http.StatusNotFound, "license not found")
		return
	}

	params, err := parseListParams(r, machineSorts)
	if err != nil {
		if writeListParamError(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "invalid list parameters")
		return
	}

	listParams := store.MachineListParams{
		ListParams: store.ListParams{
			Search: params.Search,
			Sort:   params.Sort,
			Order:  params.Order,
			Limit:  params.Limit,
			Offset: params.Offset,
		},
		LicenseID: licenseID,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		switch status {
		case "active", "released":
			listParams.Status = status
		default:
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
	}

	machines, total, err := s.store.ListLicenseMachines(r.Context(), listParams)
	if err != nil {
		writeInternalError(w, r, err, "failed to list machines")
		return
	}

	resp := make([]machineResponse, 0, len(machines))
	for _, m := range machines {
		resp = append(resp, machineToResponse(&m))
	}

	writeJSON(w, http.StatusOK, newPageResponse(resp, params.Page, params.PageSize, total))
}

func (s *Server) handleUpdateLicenseMachine(w http.ResponseWriter, r *http.Request) {
	licenseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	machineID, err := uuid.Parse(chi.URLParam(r, "machineId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid machine id")
		return
	}

	var req updateMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	machine, err := s.store.UpdateMachineName(r.Context(), licenseID, machineID, req.Name)
	if err != nil {
		writeInternalError(w, r, err, "failed to update machine")
		return
	}
	if machine == nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	auditResource(r.Context(), machine.ID, store.MachineDisplayName(machine))

	writeJSON(w, http.StatusOK, machineToResponse(machine))
}

func (s *Server) handleReleaseLicenseMachine(w http.ResponseWriter, r *http.Request) {
	licenseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	machineID, err := uuid.Parse(chi.URLParam(r, "machineId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid machine id")
		return
	}

	var byUserID *uuid.UUID
	if principal, ok := auth.PrincipalFromContext(r.Context()); ok {
		byUserID = principal.ActingUserID()
	}

	machine, err := s.store.DeactivateMachine(r.Context(), licenseID, machineID, byUserID)
	if err != nil {
		writeInternalError(w, r, err, "failed to release machine")
		return
	}
	if machine == nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	auditResource(r.Context(), machine.ID, store.MachineDisplayName(machine))

	writeJSON(w, http.StatusOK, machineToResponse(machine))
}

func parseMaxActivations(raw *int) (*int, error) {
	if raw == nil {
		return nil, nil
	}
	if *raw < 1 {
		return nil, strconv.ErrRange
	}
	return raw, nil
}
