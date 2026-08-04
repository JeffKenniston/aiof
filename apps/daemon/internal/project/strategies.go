package project

import (
	"context"
	"fmt"
	"log/slog"
)

// BaseModalityHandler provides default implementation for ModalityHandler methods.
type BaseModalityHandler struct {
	modality Modality
	logger   *slog.Logger
}

func NewBaseModalityHandler(m Modality, logger *slog.Logger) *BaseModalityHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &BaseModalityHandler{modality: m, logger: logger}
}

func (h *BaseModalityHandler) Modality() Modality {
	return h.modality
}

func (h *BaseModalityHandler) Initialize(ctx context.Context, p *Project) error {
	h.logger.Info("Initializing project modality handler", slog.String("project_id", p.Metadata.ID), slog.String("modality", string(h.modality)))
	return nil
}

func (h *BaseModalityHandler) ProvisionSkills(ctx context.Context, p *Project) ([]string, error) {
	def := GetDefaultConfigForModality(h.modality)
	return def.ActiveSkills, nil
}

func (h *BaseModalityHandler) Teardown(ctx context.Context, p *Project) error {
	h.logger.Info("Tearing down project modality handler", slog.String("project_id", p.Metadata.ID), slog.String("modality", string(h.modality)))
	return nil
}

// ModalityHandlerRegistry manages the registry of concrete modality strategy handlers.
type ModalityHandlerRegistry struct {
	handlers map[Modality]ModalityHandler
}

// NewModalityHandlerRegistry initializes the registry with all 10 default modality handlers.
func NewModalityHandlerRegistry(logger *slog.Logger) *ModalityHandlerRegistry {
	r := &ModalityHandlerRegistry{
		handlers: make(map[Modality]ModalityHandler),
	}
	for _, m := range AllModalities {
		r.handlers[m] = NewBaseModalityHandler(m, logger)
	}
	return r
}

// GetHandler retrieves the strategy handler for a given modality.
func (r *ModalityHandlerRegistry) GetHandler(m Modality) (ModalityHandler, error) {
	h, exists := r.handlers[m]
	if !exists {
		return nil, fmt.Errorf("project: no strategy handler registered for modality '%s'", m)
	}
	return h, nil
}

// RegisterHandler registers a custom or overridden modality strategy handler.
func (r *ModalityHandlerRegistry) RegisterHandler(h ModalityHandler) {
	r.handlers[h.Modality()] = h
}
