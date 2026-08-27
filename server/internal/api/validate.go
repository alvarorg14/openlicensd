package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/openlicensd/openlicensd/server/internal/license"
	"github.com/openlicensd/openlicensd/server/internal/store"
)

type validateRequest struct {
	Key         string `json:"key"`
	Product     string `json:"product"`
	Fingerprint string `json:"fingerprint"`
	Hostname    string `json:"hostname"`
}

type registryCredentialsResponse struct {
	Registry  string `json:"registry"`
	Username  string `json:"username"`
	Secret    string `json:"secret"`
	ExpiresAt int64  `json:"expires_at"`
}

func (s *Server) resolveValidLicense(ctx context.Context, rawKey, requestedProduct, fingerprint, hostname, clientIP string) (*store.License, license.ValidationResult, error) {
	keyHash := license.HashKey(rawKey)
	lic, err := s.store.GetLicenseByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, license.ValidationResult{}, err
	}

	if lic == nil {
		return nil, license.ValidationResult{Valid: false, Reason: "not_found"}, nil
	}

	now := time.Now()

	if lic.ExpirationBasis == store.ExpirationOnFirstValidation &&
		lic.ActivatedAt == nil &&
		lic.ExpiresAt == nil &&
		lic.DurationDays != nil {
		expiresAt := license.ComputeExpiry(lic.DurationDays, now)
		if expiresAt != nil {
			activated, err := s.store.ActivateLicense(ctx, lic.ID, *expiresAt)
			if err != nil {
				return nil, license.ValidationResult{}, err
			}
			if activated != nil {
				lic = activated
			}
		}
	}

	_ = s.store.RecordValidation(ctx, lic.ID)

	result := license.Validate(
		lic.ExpiresAt,
		lic.GracePeriodDays,
		lic.Revoked,
		requestedProduct,
		lic.ProductCode,
		now,
	)
	result.Policy = lic.PolicyName

	if !result.Valid {
		return lic, result, nil
	}

	fingerprint = store.SanitizeFingerprint(fingerprint)
	hostname = store.SanitizeHostname(hostname)

	if lic.MaxActivations != nil {
		if fingerprint == "" {
			result.Valid = false
			result.Reason = "fingerprint_required"
			s.setActivationFields(&result, lic)
			return lic, result, nil
		}

		machine, allowed, err := s.store.RecordActivation(ctx, lic.ID, fingerprint, hostname, clientIP, lic.MaxActivations)
		if err != nil {
			return nil, license.ValidationResult{}, err
		}
		if !allowed {
			result.Valid = false
			result.Reason = "activation_limit"
			s.setActivationFields(&result, lic)
			return lic, result, nil
		}
		if machine != nil {
			lic.ActivationCount, err = s.store.CountActiveMachines(ctx, lic.ID)
			if err != nil {
				return nil, license.ValidationResult{}, err
			}
		}
	} else if fingerprint != "" {
		_, allowed, err := s.store.RecordActivation(ctx, lic.ID, fingerprint, hostname, clientIP, nil)
		if err != nil {
			return nil, license.ValidationResult{}, err
		}
		if !allowed {
			result.Valid = false
			result.Reason = "activation_limit"
			s.setActivationFields(&result, lic)
			return lic, result, nil
		}
		lic.ActivationCount, err = s.store.CountActiveMachines(ctx, lic.ID)
		if err != nil {
			return nil, license.ValidationResult{}, err
		}
	}

	s.setActivationFields(&result, lic)
	return lic, result, nil
}

func (s *Server) setActivationFields(result *license.ValidationResult, lic *store.License) {
	if lic.ActivationCount > 0 || lic.MaxActivations != nil {
		count := lic.ActivationCount
		result.ActivationCount = &count
	}
	if lic.MaxActivations != nil {
		max := *lic.MaxActivations
		result.MaxActivations = &max
	}
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	clientIP := s.clientIP.From(r)
	_, result, err := s.resolveValidLicense(r.Context(), req.Key, req.Product, req.Fingerprint, req.Hostname, clientIP)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate license")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRegistryCredentials(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	clientIP := s.clientIP.From(r)
	lic, result, err := s.resolveValidLicense(r.Context(), req.Key, req.Product, req.Fingerprint, req.Hostname, clientIP)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate license")
		return
	}

	if !result.Valid {
		reason := result.Reason
		if reason == "" {
			reason = "invalid"
		}
		writeError(w, http.StatusForbidden, reason)
		return
	}

	creds, err := s.harbor.CreateEphemeralRobot(
		r.Context(),
		s.cfg.Harbor.Projects,
		s.cfg.Harbor.RobotDurationDays,
		s.cfg.Harbor.RobotNamePrefix,
		lic.KeyPrefix,
	)
	if err != nil {
		log.Printf("registry credentials: harbor create robot failed: %v", err)
		message := "failed to issue registry credentials"
		if s.cfg.Harbor.Debug {
			message = fmt.Sprintf("%s: %v", message, err)
		}
		writeError(w, http.StatusBadGateway, message)
		return
	}

	if err := s.harbor.CleanupExpiredRobots(r.Context(), s.cfg.Harbor.RobotNamePrefix); err != nil {
		log.Printf("registry credentials: harbor cleanup failed: %v", err)
	}

	writeJSON(w, http.StatusOK, registryCredentialsResponse{
		Registry:  s.harbor.RegistryHost(),
		Username:  creds.Name,
		Secret:    creds.Secret,
		ExpiresAt: creds.ExpiresAt,
	})
}
