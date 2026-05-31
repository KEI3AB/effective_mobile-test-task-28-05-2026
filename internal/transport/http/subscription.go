package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/effective_mobile-test-task-28-05-2026/internal/domain"
	"github.com/effective_mobile-test-task-28-05-2026/internal/usecase"
	"github.com/go-chi/chi/v5"
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
	log     *slog.Logger
}

func NewSubscriptionHandler(uc SubscriptionUseCase, log *slog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{usecase: uc, log: log}
}

// CreateSubscription godoc
// @Summary      Создание новой подписки
// @Description  Создает запись о подписке пользователя на определенный сервис
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        request body CreateSubscriptionRequest true "Данные о подписке"
// @Success      201 {object} CreateSubscriptionResponse "Успешно создано"
// @Failure      400 {object} ErrorResponse "Неверный формат запроса или ошибка валидации"
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router       /subscriptions [post]
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

// GetSubscription godoc
// @Summary      Получение подписки по ID
// @Description  Возвращает детальную информацию о конкретной подписке
// @Tags         subscriptions
// @Produce      json
// @Param        id path string true "UUID подписки"
// @Success      200 {object} ReadSubscriptionResponse "Успешное получение"
// @Failure      400 {object} ErrorResponse "Неверный формат ID"
// @Failure      404 {object} ErrorResponse "Подписка не найдена"
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router       /subscriptions/{id} [get]
func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	qpSubID := chi.URLParam(r, "id")

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

// UpdateSubscription godoc
// @Summary      Обновление подписки
// @Description  Полностью обновляет данные существующей подписки
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id path string true "UUID подписки"
// @Param        request body UpdateSubscriptionRequest true "Новые данные подписки"
// @Success      200 "Подписка успешно обновлена"
// @Failure      400 {object} ErrorResponse "Неверный формат запроса или ошибка валидации"
// @Failure      404 {object} ErrorResponse "Подписка не найдена"
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router       /subscriptions/{id} [put]
func (h *SubscriptionHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	qpSubID := chi.URLParam(r, "id")
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

// DeleteSubscription godoc
// @Summary      Удаление подписки
// @Description  Удаляет запись о подписке по ее ID
// @Tags         subscriptions
// @Param        id path string true "UUID подписки"
// @Success      204 "Успешно удалено (без тела ответа)"
// @Failure      400 {object} ErrorResponse "Неверный формат ID"
// @Failure      404 {object} ErrorResponse "Подписка не найдена"
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router       /subscriptions/{id} [delete]
func (h *SubscriptionHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	qpSubID := chi.URLParam(r, "id")
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

// ListSubscriptions godoc
// @Summary      Получение списка подписок
// @Description  Возвращает список подписок пользователя с поддержкой пагинации
// @Tags         subscriptions
// @Produce      json
// @Param        user_id query string true "UUID пользователя"
// @Param        limit query int false "Количество записей (по умолчанию 20, макс 100)"
// @Param        offset query int false "Смещение для пагинации"
// @Success      200 {object} ListSubscriptionsResponse "Успешное получение списка"
// @Failure      400 {object} ErrorResponse "Неверный формат параметров"
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router       /subscriptions [get]
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

// CalculateTotalCost godoc
// @Summary      Подсчет суммарной стоимости
// @Description  Рассчитывает общую сумму трат на подписки за выбранный период с возможностью фильтрации
// @Tags         calculations
// @Produce      json
// @Param        user_id query string false "UUID пользователя (опционально)"
// @Param        service_name query string false "Название сервиса (опционально)"
// @Param        start_date query string true "Начало периода в формате MM-YYYY"
// @Param        end_date query string false "Конец периода в формате MM-YYYY (опционально)"
// @Success      200 {object} CalcTotalCostResponse "Успешный расчет"
// @Failure      400 {object} ErrorResponse "Неверный формат дат или параметров"
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router       /costs [get]
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
