package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
)

func main() {
	config.LoadEnv()
	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	secret := config.GetEnv("JWT_SECRET", "")
	if secret == "" {
		log.Fatal("JWT_SECRET kosong: jalankan dari folder proyek yang berisi .env")
	}
	jwtManager := helper.NewJWTManager(
		secret,
		config.GetEnv("JWT_ISSUER", "api-students"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	print := func(username, varName string) {
		var u model.User
		err := pool.QueryRow(ctx,
			`SELECT id, username, email, role, is_active FROM users WHERE username = $1`,
			username).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive)
		if err != nil {
			log.Fatalf("membaca user %s: %v", username, err)
		}
		t, err := jwtManager.GenerateAccess(u)
		if err != nil {
			log.Fatalf("membuat token %s: %v", username, err)
		}
		fmt.Printf("export %s=%q\n", varName, t)
	}
	print("admin", "ADMIN_TOKEN")
	print("mhs001", "MHS_TOKEN")
	print("sari", "USER_TOKEN")
}
