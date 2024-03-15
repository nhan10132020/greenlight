package data

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Permissions pq.StringArray

func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}

	return false
}

type PermissionsModel struct {
	DB *gorm.DB
}

func (m PermissionsModel) GetAllForUser(userID int64) (Permissions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var permissions Permissions

	if err := m.DB.
		WithContext(ctx).
		Table("permissions").
		Joins("inner join users_permissions ON users_permissions.permission_id = permissions.id").
		Where("user_id = ?", userID).
		Select("code").Find(&permissions).Error; err != nil {

		return nil, err
	}

	return permissions, nil
}

func (m PermissionsModel) AddForUser(userID int64, codes ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.DB.
		WithContext(ctx).
		Exec("INSERT INTO users_permissions SELECT ?, permissions.id FROM permissions WHERE permissions.code=ANY(?)", userID, pq.StringArray(codes)).
		Error; err != nil {
		return err
	}

	fmt.Println()

	return nil
}
