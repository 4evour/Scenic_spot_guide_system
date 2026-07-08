package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *model.User) error
	FindByID(id uint) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	FindByGuestToken(guestToken string) (*model.User, error)
	FindAll() ([]model.User, error)
	FindAllPaginated(page, pageSize int) ([]model.User, int64, error)
	FindByRole(role string) ([]model.User, error)
	Update(user *model.User) error
	UpdateFields(id uint, fields map[string]interface{}) error
	UpgradeGuest(userID uint, fields map[string]interface{}) error
	Delete(id uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepository) Update(user *model.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&model.User{}, user.ID).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
			"username":            user.Username,
			"password":            user.Password,
			"email":               user.Email,
			"role":                user.Role,
			"preferred_avatar_id": user.PreferredAvatarID,
		}).Error
	})
}

func (r *userRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&model.User{}, id).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", id).Updates(fields).Error
	})
}

func (r *userRepository) FindAll() ([]model.User, error) {
	var users []model.User
	err := r.db.Find(&users).Error
	return users, err
}

func (r *userRepository) FindAllPaginated(page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	if err := r.db.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
	return users, total, err
}

func (r *userRepository) FindByRole(role string) ([]model.User, error) {
	var users []model.User
	err := r.db.Where("role = ?", role).Find(&users).Error
	return users, err
}

func (r *userRepository) FindByGuestToken(guestToken string) (*model.User, error) {
	var user model.User
	err := r.db.Where("guest_token = ?", guestToken).First(&user).Error
	return &user, err
}

func (r *userRepository) UpgradeGuest(userID uint, fields map[string]interface{}) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		// 清除游客标识，更新为正式用户
		fields["guest_token"] = ""
		fields["display_name"] = ""
		return tx.Model(&user).Updates(fields).Error
	})
}

func (r *userRepository) Delete(id uint) error {
	result := r.db.Delete(&model.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
