package handlers

type HealthResponse struct {
	Status   string `json:"status" example:"healthy"`
	Postgres string `json:"postgres" example:"up"`
	Redis    string `json:"redis" example:"up"`
}
