package repositories

import (
	"awesomeproject/backend/internal/models"

	"gorm.io/gorm"
)

type FriendRepository struct {
	db *gorm.DB
}

func NewFriendRepository(db *gorm.DB) *FriendRepository {
	return &FriendRepository{db: db}
}

func (r *FriendRepository) Exists(userID, friendID uint64) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *FriendRepository) CreatePair(userID, friendID uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		rows := []models.Friendship{
			{UserID: userID, FriendID: friendID},
			{UserID: friendID, FriendID: userID},
		}
		return tx.Create(&rows).Error
	})
}

func (r *FriendRepository) ListByUser(userID uint64) ([]models.Friendship, error) {
	var rows []models.Friendship
	if err := r.db.Where("user_id = ?", userID).Order("friend_id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
