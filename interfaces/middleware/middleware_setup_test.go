package middleware

import (
	"fmt"
	"go-rest-skeleton/domain/entity"
	"go-rest-skeleton/domain/repository"
	"go-rest-skeleton/infrastructure/authorization"
	"go-rest-skeleton/infrastructure/config"
	"go-rest-skeleton/infrastructure/persistence"
	"go-rest-skeleton/infrastructure/util"
	"log"

	"github.com/bxcodec/faker"

	"github.com/go-redis/redis/v8"

	"github.com/jinzhu/gorm"
	"github.com/joho/godotenv"
)

// Repositories represent it self.
type Repositories struct {
	Permission repository.PermissionRepository
	Role       repository.RoleRepository
	User       repository.UserRepository
	db         *gorm.DB
}

// RedisService represent it self.
type RedisService struct {
	Auth   authorization.AuthInterface
	Client *redis.Client
}

// Dependencies represent it self.
type Dependencies struct {
	db *gorm.DB
	rd *RedisService
	ag *authorization.Gateway
	at *authorization.Token
	cf *config.Config
}

func InitConfig() *config.Config {
	if err := godotenv.Load(fmt.Sprintf("%s/.env", util.RootDir())); err != nil {
		log.Println("no .env file provided")
	}

	return config.New()
}

func Setup() *Dependencies {
	conf := InitConfig()
	db, _ := DBConnSetup(conf.DBTestConfig)
	rd, _ := RedisConnSetup(conf.RedisConfig)
	dbServices, _ := DBServicesSetup(db)
	redisServices, _ := RedisServicesSetup(rd)

	authBasic := authorization.NewBasicAuth(dbServices.User)
	authJWT := authorization.NewJWTAuth(conf.KeyConfig, redisServices.Client)
	authGateway := authorization.NewAuthGateway(authBasic, authJWT, dbServices.User, dbServices.Role)
	authToken := authorization.NewToken(conf.KeyConfig, redisServices.Client)

	return &Dependencies{
		db: dbServices.db,
		rd: redisServices,
		ag: authGateway,
		at: authToken,
		cf: conf,
	}
}

// DBConnSetup is a function.
func DBConnSetup(config config.DBTestConfig) (*gorm.DB, error) {
	dbURL := ""
	switch config.DBDriver {
	case "postgres":
		dbURL = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s",
			config.DBHost,
			config.DBPort,
			config.DBUser,
			config.DBName,
			config.DBPassword,
		)
	case "mysql":
		dbURL = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
			config.DBUser,
			config.DBPassword,
			config.DBHost,
			config.DBPort,
			config.DBName,
		)
	}

	db, err := gorm.Open(config.DBDriver, dbURL)
	if err != nil {
		return nil, err
	}
	db.LogMode(false)

	err = db.AutoMigrate(
		&entity.Module{},
		&entity.Permission{},
		&entity.Role{},
		&entity.RolePermission{},
		&entity.User{},
		&entity.UserRole{},
	).Error
	if err != nil {
		return nil, err
	}
	return db, nil
}

// DBServicesSetup will initialize db connection and return repositories.
func DBServicesSetup(db *gorm.DB) (*Repositories, error) {
	return &Repositories{
		Permission: persistence.NewPermissionRepository(db),
		Role:       persistence.NewRoleRepository(db),
		User:       persistence.NewUserRepository(db),
		db:         db,
	}, nil
}

// RedisConnSetup will initialize redis connection and return redis client.
func RedisConnSetup(config config.RedisConfig) (*redis.Client, error) {
	return redis.NewClient(&redis.Options{
		Addr:     config.RedisHost + ":" + config.RedisPort,
		Password: config.RedisPassword,
		DB:       10,
	}), nil
}

// RedisServiceSetup will initialize connection to redis server.
func RedisServicesSetup(rc *redis.Client) (*RedisService, error) {
	return &RedisService{
		Auth:   authorization.NewAuth(rc),
		Client: rc,
	}, nil
}

func seedUser(db *gorm.DB) (*entity.User, *entity.UserFaker, error) {
	userFaker := entity.UserFaker{}
	_ = faker.FakeData(&userFaker)
	user := entity.User{
		FirstName: userFaker.FirstName,
		LastName:  userFaker.LastName,
		Email:     userFaker.Email,
		Phone:     userFaker.Phone,
		Password:  userFaker.Password,
	}
	err := db.Create(&user).Error
	if err != nil {
		return nil, nil, err
	}
	return &user, &userFaker, nil
}