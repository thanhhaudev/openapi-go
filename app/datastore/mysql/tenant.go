package mysql

import (
	"github.com/thanhhaudev/openapi-go/app/model"
	"github.com/thanhhaudev/openapi-go/app/repository"
	"gorm.io/gorm"
)

type tenantRepository struct {
	gorm *gorm.DB
}

func (t tenantRepository) FindByApiKey(apiKey string) (*model.Tenant, error) {
	tenant := &model.Tenant{}

	err := t.gorm.
		Where(`api_key = ?`, apiKey).
		First(tenant).Error
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

// Find finds a tenant by app key and app secret
func (t tenantRepository) Find(appKey, appSecret string) (*model.Tenant, error) {
	tenant := &model.Tenant{}

	err := t.gorm.
		Where(`api_key = ? AND api_secret = ?`, appKey, appSecret).
		First(tenant).Error
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(gorm *gorm.DB) repository.TenantRepository {
	return &tenantRepository{
		gorm: gorm,
	}
}
