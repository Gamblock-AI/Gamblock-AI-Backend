package service

import (
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type Container struct {
	Auth                 *AuthService
	Device               *DeviceService
	Accountability       *AccountabilityService
	AccountabilityGroups *AccountabilityGroupService
	Admin                *AdminService
	Support              *SupportService
	Reflection           *ReflectionService
	Organization         *OrganizationService
	Mission              *MissionService
	Recovery             *RecoveryService
	Client               *ClientService
	Education            *EducationService
}

func NewContainer(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *Container {
	whatsapp := NewWhatsAppService(cfg, logger)

	return &Container{
		Auth:                 NewAuthService(repo, cfg, logger),
		Device:               NewDeviceService(repo, logger),
		Accountability:       NewAccountabilityService(repo, cfg, whatsapp, logger),
		AccountabilityGroups: NewAccountabilityGroupService(repo, cfg),
		Admin:                NewAdminServiceWithConfig(repo, cfg, logger),
		Support:              NewSupportServiceWithConfig(repo, cfg, logger),
		Reflection:           NewReflectionService(repo, cfg, logger),
		Organization:         NewOrganizationService(repo, logger),
		Mission:              NewMissionService(repo, logger),
		Recovery:             NewRecoveryServiceWithConfig(repo, cfg),
		Client:               NewClientService(repo, cfg),
		Education:            NewEducationService(repo, cfg),
	}
}
