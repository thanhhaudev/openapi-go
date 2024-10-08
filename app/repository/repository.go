package repository

import "github.com/thanhhaudev/go-open-api/app/model"

type (
	TenantRepository interface {
		Find(appKey, appSecret string) (*model.Tenant, error)
		FindByApiKey(apiKey string) (*model.Tenant, error)
	}

	UserRepository interface {
		FindAll() ([]*model.User, error)
		FindByID(id uint) (*model.User, error)
		FindByIDs(ids []uint) ([]*model.User, error)
		FindByEmail(email string) (*model.User, error)
		Create(user *model.User) error
		Update(user *model.User) error
		Delete(user *model.User) error
	}

	MessageRepository interface {
		FindByID(id uint) (*model.Message, error)
		Create(message *model.Message) error
		Update(message *model.Message) error
		Delete(id uint) error
	}

	UserMessageRepository interface {
		FindByUserID(userId uint) ([]*model.UserMessage, error)
		FindByID(userId, id uint) (*model.UserMessage, error)
		Create(userMessage *model.UserMessage) error
		Update(userMessage *model.UserMessage) error
		Delete(userId, id uint) error
	}
)
