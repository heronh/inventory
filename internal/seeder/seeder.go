package seeder

import (
	"fmt"

	"inventory/internal/config"
	"inventory/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB, cfg *config.Config) error {
	roles := []string{cfg.RoleSeederOne, cfg.RoleSeederTwo, cfg.RoleSeederThree}
	for _, roleName := range roles {
		if roleName == "" {
			continue
		}

		role := models.Privilege{Description: roleName}
		if err := db.Where(models.Privilege{Description: roleName}).FirstOrCreate(&role).Error; err != nil {
			return err
		}
	}

	var role models.Privilege
	if err := db.Where("description = ?", cfg.UserSeederRole).First(&role).Error; err != nil {
		return fmt.Errorf("seeder role not found: %w", err)
	}

	var existing models.User
	err := db.Where("email = ?", cfg.UserSeederEmail).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.UserSeederPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := models.User{
		Name:        cfg.UserSeederName,
		Email:       cfg.UserSeederEmail,
		Password:    string(hashed),
		PrivilegeID: role.ID,
	}

	return db.Create(&user).Error
}
