package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/effective_mobile-test-task-28-05-2026/internal/domain"
	"github.com/effective_mobile-test-task-28-05-2026/internal/usecase"
	"github.com/google/uuid"
	"github.com/mailru/easyjson"
)

type SubscriptionUseCase interface {
	Create(ctx context.Context, sub domain.Subscription) (uuid.UUID, error)
	Read(ctx context.Context, subID uuid.UUID) (domain.Subscription, error)
	Update(ctx context.Context, sub domain.Subscription) error
	Delete(ctx context.Context, subID uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Subscription, error)
	CalculatePriceFromPeriod(ctx context.Context, userID uuid.UUID, serviceName string, periodStart time.Time, periodEnd *time.Time) (int, error)
}

type SubscriptionHandler struct {
	usecase SubscriptionUseCase
}

func NewSubscriptionHandler(uc SubscriptionUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{usecase: uc}
}

func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateSubscriptionRequest
	if err := easyjson.UnmarshalFromReader(r.Body, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	defer r.Body.Close()

	startDate, err := DTODateToDomainDate(req.StartDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid start date format (expected MM-YYYY)")
		return
	}

	var endDatePtr *time.Time
	if req.EndDate != "" {
		parsedDate, err := DTODateToDomainDate(req.EndDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid end date format (expected MM-YYYY)")
			return
		}
		endDatePtr = &parsedDate
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	domainSub := domain.Subscription{
		UserID:      userID,
		ServiceName: req.ServiceName,
		Price:       req.Price,
		StartDate:   startDate,
		EndDate:     endDatePtr,
	}

	subID, err := h.usecase.Create(ctx, domainSub)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidPrice) || errors.Is(err, usecase.ErrInvalidDates) || errors.Is(err, usecase.ErrEmptyServiceName) {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	resp := CreateSubscriptionResponse{SubscriptionID: subID.String()}
	payload, err := easyjson.Marshal(resp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
		return
	}

	respondWithJSON(w, http.StatusCreated, payload)
}

func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	qpSubID := r.URL.Query().Get("id")

	subID, err := uuid.Parse(qpSubID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid subscription_id format")
		return
	}

	sub, err := h.usecase.Read(ctx, subID)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Subscription not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	startDate := sub.StartDate.Format("01-2006")
	var endDate string
	if sub.EndDate != nil {
		endDate = sub.EndDate.Format("01-2006")
	}

	resp := ReadSubscriptionResponse{
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		UserID:      sub.UserID.String(),
		StartDate:   startDate,
		EndDate:     endDate,
	}
	payload, err := easyjson.Marshal(resp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
		return
	}

	respondWithJSON(w, http.StatusOK, payload)
}

func (h *SubscriptionHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	qpSubID := r.URL.Query().Get("id")
	subID, err := uuid.Parse(qpSubID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid subscription_id format")
		return
	}

	var req UpdateSubscriptionRequest
	if err := easyjson.UnmarshalFromReader(r.Body, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	defer r.Body.Close()

	startDate, err := DTODateToDomainDate(req.StartDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid start date format")
		return
	}

	var endDatePtr *time.Time
	if req.EndDate != "" {
		parsedDate, err := DTODateToDomainDate(req.EndDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid end date format")
			return
		}
		endDatePtr = &parsedDate
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	domainSub := domain.Subscription{
		ID:          subID,
		UserID:      userID,
		ServiceName: req.ServiceName,
		Price:       req.Price,
		StartDate:   startDate,
		EndDate:     endDatePtr,
	}

	if err := h.usecase.Update(ctx, domainSub); err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Subscription not found")
			return
		}
		if errors.Is(err, usecase.ErrInvalidPrice) || errors.Is(err, usecase.ErrInvalidDates) || errors.Is(err, usecase.ErrEmptyServiceName) {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SubscriptionHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	qpSubID := r.URL.Query().Get("id")
	subID, err := uuid.Parse(qpSubID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid subscription_id format")
		return
	}

	if err := h.usecase.Delete(ctx, subID); err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Subscription not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userIDStr := r.URL.Query().Get("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	subs, err := h.usecase.List(ctx, userID, limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	dtoSubs := make([]SubscriptionDTO, 0, len(subs))
	for _, sub := range subs {
		endDateStr := ""
		if sub.EndDate != nil {
			endDateStr = sub.EndDate.Format("01-2006")
		}

		dtoSubs = append(dtoSubs, SubscriptionDTO{
			ServiceName: sub.ServiceName,
			Price:       sub.Price,
			UserID:      sub.UserID.String(),
			StartDate:   sub.StartDate.Format("01-2006"),
			EndDate:     endDateStr,
		})
	}

	resp := ListSubscriptionsResponse{Subscriptions: dtoSubs}
	payload, err := easyjson.Marshal(resp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
		return
	}

	respondWithJSON(w, http.StatusOK, payload)
}

func (h *SubscriptionHandler) CalculateTotalCost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userIDStr := r.URL.Query().Get("user_id")
	serviceName := r.URL.Query().Get("service_name")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	if startDateStr == "" {
		respondWithError(w, http.StatusBadRequest, "start_date is required")
		return
	}

	periodStart, err := DTODateToDomainDate(startDateStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid start date format")
		return
	}

	var periodEndPtr *time.Time
	if endDateStr != "" {
		parsedEnd, err := DTODateToDomainDate(endDateStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid end date format")
			return
		}
		periodEndPtr = &parsedEnd
	}

	var userID uuid.UUID
	if userIDStr != "" {
		parsedUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid user_id format")
			return
		}
		userID = parsedUUID
	}

	totalCost, err := h.usecase.CalculatePriceFromPeriod(ctx, userID, serviceName, periodStart, periodEndPtr)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidDates) {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	resp := CalcTotalCostResponse{TotalCost: totalCost}
	payload, err := easyjson.Marshal(resp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
		return
	}

	respondWithJSON(w, http.StatusOK, payload)
}

func DTODateToDomainDate(date string) (time.Time, error) {
	parsedTime, err := time.Parse("01-2006", date)
	if err != nil {
		return time.Time{}, err
	}

	return parsedTime, nil
}

func respondWithJSON(w http.ResponseWriter, code int, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(payload)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	resp := ErrorResponse{Error: message}
	payload, _ := easyjson.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(payload)
}
