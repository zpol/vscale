package store

import (
	"fmt"
	"log"
	"time"

	"github.com/zpol/vscale/internal/auth"
	"github.com/zpol/vscale/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for i := 0; i < 30; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			sqlDB, e := db.DB()
			if e == nil {
				sqlDB.SetMaxOpenConns(25)
				sqlDB.SetMaxIdleConns(5)
				sqlDB.SetConnMaxLifetime(time.Hour)
			}
			return db, nil
		}
		log.Printf("db connect attempt %d: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("database unreachable: %w", err)
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Vscale{},
		&models.Hub{},
		&models.Device{},
		&models.Membership{},
		&models.JoinToken{},
		&models.Route{},
		&models.IpAllocation{},
		&models.MetricSnapshot{},
	)
}

func EnsureAdmin(db *gorm.DB, username, password, email string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	var u models.User
	err = db.Where("username = ?", username).First(&u).Error
	if err == nil {
		u.Password = hash
		u.Email = email
		u.Role = models.RoleAdmin
		return db.Save(&u).Error
	}
	u = models.User{
		Username: username,
		Password: hash,
		Email:    email,
		Role:     models.RoleAdmin,
	}
	return db.Create(&u).Error
}
