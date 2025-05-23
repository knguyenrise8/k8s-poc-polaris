package handlers

import "k8s-web-service/internal/config"

// Handler contains the application dependencies
type Handler struct {
	config *config.Config
}

// New creates a new handler instance
func New(cfg *config.Config) *Handler {
	return &Handler{config: cfg}
}
