package dsDb

import (
	"ai-chatbot/model"
	"context"

	"gorm.io/gorm"
)

type DsMessageDao struct {
	ctx context.Context
	db  *gorm.DB
}

func NewDsMessageDao(ctx context.Context, db *gorm.DB) *DsMessageDao {
	return &DsMessageDao{ctx: ctx, db: db}
}

func (d *DsMessageDao) Create(user *model.DsMessage) error {
	return d.db.WithContext(d.ctx).Create(user).Error
}

func (d *DsMessageDao) BatchCreate(users []*model.DsMessage) error {
	return d.db.WithContext(d.ctx).Create(&users).Error
}
