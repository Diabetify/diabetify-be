package utils

import (
	"fmt"
	"log"
)

// ==================== ORIGINAL FUNCTIONS (Single Database) ====================

// CheckForDuplicateEmails checks if any of the test emails already exist in the database
func CheckForDuplicateEmails(startIndex, endIndex int) error {
	db, err := connectToSingleDatabase()
	if err != nil {
		return err
	}

	log.Println("Connected to database successfully")
	log.Printf("Checking for duplicate emails in range %d-%d...", startIndex, endIndex)

	// Check for duplicate emails using a more efficient query
	var duplicateEmails []string
	err = db.Raw(`
		SELECT email 
		FROM users 
		WHERE email LIKE 'testuser%@example.com'
		  AND email BETWEEN ? AND ?
		GROUP BY email 
		HAVING COUNT(*) > 1
		ORDER BY email
	`, fmt.Sprintf("testuser%d@example.com", startIndex), fmt.Sprintf("testuser%d@example.com", endIndex)).Scan(&duplicateEmails).Error

	if err != nil {
		return fmt.Errorf("failed to check for duplicate emails: %v", err)
	}

	if len(duplicateEmails) > 0 {
		log.Printf("Found %d duplicate emails:", len(duplicateEmails))
		for _, email := range duplicateEmails {
			log.Printf("  - %s", email)
		}
	} else {
		log.Println("No duplicate emails found")
	}

	log.Println("Email duplicate check completed")
	return nil
}

// DeleteTestUsers deletes test users in the specified range
func DeleteTestUsers(startIndex, endIndex int) error {
	db, err := connectToSingleDatabase()
	if err != nil {
		return err
	}

	log.Println("Connected to database successfully")
	log.Printf("Deleting test users with emails in range testuser%d@example.com to testuser%d@example.com...",
		startIndex, endIndex)

	// Delete test users in batches for better performance
	batchSize := 1000
	totalDeleted := 0

	for i := startIndex; i <= endIndex; i += batchSize {
		end := i + batchSize - 1
		if end > endIndex {
			end = endIndex
		}

		// Generate list of emails to delete
		var emails []string
		for j := i; j <= end; j++ {
			emails = append(emails, fmt.Sprintf("testuser%d@example.com", j))
		}

		// Delete users with these emails
		result := db.Table("users").Where("email IN ?", emails).Delete(nil)
		if result.Error != nil {
			return fmt.Errorf("failed to delete users batch %d-%d: %v", i, end, result.Error)
		}

		totalDeleted += int(result.RowsAffected)
		log.Printf("Deleted %d users in range %d-%d", result.RowsAffected, i, end)
	}

	log.Printf("Successfully deleted %d test users", totalDeleted)
	return nil
}
