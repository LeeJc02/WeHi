package repositories

import (
	"awesomeproject/backend/internal/models"

	"gorm.io/gorm"
)

type ConversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) CreateConversation(conversation *models.Conversation, members []models.ConversationMember) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conversation).Error; err != nil {
			return err
		}
		for index := range members {
			members[index].ConversationID = conversation.ID
		}
		return tx.Create(&members).Error
	})
}

func (r *ConversationRepository) FindByID(id uint64) (*models.Conversation, error) {
	var conversation models.Conversation
	if err := r.db.First(&conversation, id).Error; err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *ConversationRepository) FindDirect(userA, userB uint64) (*models.Conversation, error) {
	var conversation models.Conversation
	err := r.db.Table("conversations c").
		Select("c.*").
		Joins("JOIN conversation_members cm ON cm.conversation_id = c.id").
		Where("c.type = ?", "direct").
		Where("cm.user_id IN ?", []uint64{userA, userB}).
		Group("c.id").
		Having("COUNT(DISTINCT cm.user_id) = 2").
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *ConversationRepository) ListByUser(userID uint64) ([]models.Conversation, error) {
	var conversations []models.Conversation
	err := r.db.Table("conversations c").
		Select("c.*").
		Joins("JOIN conversation_members cm ON cm.conversation_id = c.id").
		Where("cm.user_id = ?", userID).
		Order("c.updated_at desc, c.id desc").
		Find(&conversations).Error
	return conversations, err
}

func (r *ConversationRepository) ListMembers(conversationID uint64) ([]models.ConversationMember, error) {
	var members []models.ConversationMember
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("id asc").
		Find(&members).Error
	return members, err
}

func (r *ConversationRepository) FindMember(conversationID, userID uint64) (*models.ConversationMember, error) {
	var member models.ConversationMember
	if err := r.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *ConversationRepository) SaveMember(member *models.ConversationMember) error {
	return r.db.Save(member).Error
}

func (r *ConversationRepository) TouchConversation(conversationID uint64) error {
	return r.db.Model(&models.Conversation{}).Where("id = ?", conversationID).Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
