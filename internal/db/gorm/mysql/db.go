package mysql

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"audio_compression/config"
)

func NewDB(cfg config.MYSQL) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Dbname)
	// dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s post=%s", cfg.Host, cfg.Username, cfg.Password, cfg.Dbname, cfg.Port)

	db, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn}), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("unable to open db connection")
	}

	err = db.Use(otelgorm.NewPlugin(otelgorm.WithDBName(cfg.Dbname)))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to set gorm plugin for opentelemetry ")
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get sql db")
	}

	// Hardcode the max open connection for now
	sqlDB.SetMaxOpenConns(200)
	return db
}
