package sample

import (
	"context"
	"log"
	"time"

	"github.com/billowdev/atomic-transactors/transactors"
	"github.com/billowdev/clog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// A demonstrates bulk operations and transaction handling
func A(ctx context.Context) error {
	tx := transactors.ExtractTx(ctx)
	if tx == nil {
		clog.Panic("tx is nil")
	}
	// Create multiple users
	users := []User{
		{Name: "Alice", Email: "alice@example.com"},
		{Name: "Bob", Email: "bob@example.com"},
		{Name: "Charlie", Email: "charlie@example.com"},
	}

	// Bulk insert
	if err := tx.Create(&users).Error; err != nil {
		return err
	}

	// Bulk update - update all users created in the last hour
	if err := tx.Model(&User{}).
		Where("created_at > ?", time.Now().Add(-1*time.Hour)).
		Update("Email", "updated@example.com").Error; err != nil {
		return err
	}

	// Commit transaction
	return nil
}

// B demonstrates filtered queries and complex operations
func B(ctx context.Context) error {
	tx := transactors.ExtractTx(ctx)
	if tx == nil {
		clog.Panic("tx is nil")
	}
	// Find users with specific conditions
	var users []User
	if err := tx.Where("name LIKE ?", "A%").
		Or("email LIKE ?", "%@example.com").
		Limit(10).
		Find(&users).Error; err != nil {
		return err
	}

	// Process each user
	for _, user := range users {
		// Example: Delete users with empty email
		if user.Email == "" {
			if err := tx.Delete(&user).Error; err != nil {
				return err
			}
			continue
		}

		// Example: Update user's name if it starts with 'A'
		if len(user.Name) > 0 && user.Name[0] == 'A' {
			user.Name = "Updated_" + user.Name
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		}
	}

	// Count total users
	var count int64
	if err := tx.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}

	log.Printf("Total users after processing: %d\n", count)
	return nil
}

func main() {
	// Open SQLite database connection
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// Auto migrate schemas
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	// Example: Create a new user
	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	result := db.Create(&user)
	if result.Error != nil {
		log.Fatal("failed to create user:", result.Error)
	}

	log.Printf("Created user with ID: %d\n", user.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	txCtx := transactors.InjectTx(ctx, db)
	// Run function A
	if err := A(txCtx); err != nil {
		log.Fatal("Error in function A:", err)
	}

	// Run function B
	if err := B(txCtx); err != nil {
		log.Fatal("Error in function B:", err)
	}

	// Example: Query a user
	var queriedUser User
	db.First(&queriedUser, user.ID)
	log.Printf("Found user: %v\n", queriedUser)
}
