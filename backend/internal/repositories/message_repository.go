package repositories

import (
	"awesomeproject/backend/internal/models"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(message *models.Message) error {
	return r.db.Create(message).Error
}

func (r *MessageRepository) ListByConversation(conversationID uint64, limit int) ([]models.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var messages []models.Message
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("id desc").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

func (r *MessageRepository) CountUnread(conversationID, userID, lastReadMessageID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&models.Message{}).
		Where("conversation_id = ? AND sender_id <> ? AND id > ?", conversationID, userID, lastReadMessageID).
		Count(&count).Error
	return count, err
}

func (r *MessageRepository) FindLatest(conversationID uint64) (*models.Message, error) {
	var message models.Message
	if err := r.db.Where("conversation_id = ?", conversationID).Order("id desc").First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}
