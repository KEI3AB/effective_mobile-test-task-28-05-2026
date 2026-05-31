package http

//go:generate easyjson $GOFILE

//easyjson:json
type ErrorResponse struct {
	Error string `json:"error"`
}

//easyjson:json
type SubscriptionDTO struct {
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date,omitempty"`
}

//easyjson:json
type CreateSubscriptionRequest struct {
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date,omitempty"`
}

//easyjson:json
type CreateSubscriptionResponse struct {
	SubscriptionID string `json:"subscription_id"`
}

//easyjson:json
type ReadSubscriptionResponse struct {
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date,omitempty"`
}

//easyjson:json
type UpdateSubscriptionRequest struct {
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date,omitempty"`
}

//easyjson:json
type ListSubscriptionsResponse struct {
	Subscriptions []SubscriptionDTO `json:"subscriptions"`
}

//easyjson:json
type CalcTotalCostResponse struct {
	TotalCost int `json:"total_cost"`
}
