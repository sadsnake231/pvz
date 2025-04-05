package api

import (
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
)

type AuditHandler struct {
	service service.AuditService
}

func NewAuditHandler(service service.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}
