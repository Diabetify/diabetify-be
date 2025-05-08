package main

import (
	"diabetify/internal/utils"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	seedCmd := flag.NewFlagSet("seed", flag.ExitOnError)
	numUsers := seedCmd.Int("users", utils.DefaultNumUsers, "Number of dummy users to create")

	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
	startIndex := checkCmd.Int("start", 0, "Start index for email check")
	endIndex := checkCmd.Int("end", 1000, "End index for email check")

	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteStart := deleteCmd.Int("start", 0, "Start index for user deletion")
	deleteEnd := deleteCmd.Int("end", 1000, "End index for user deletion")

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	// Parse the subcommand
	switch os.Args[1] {
	case "seed":
		seedCmd.Parse(os.Args[2:])
		log.Printf("Starting user seeder with %d users", *numUsers)
		if err := utils.SeedUsers(*numUsers); err != nil {
			log.Fatalf("Error seeding users: %v", err)
		}

	case "check":
		checkCmd.Parse(os.Args[2:])
		log.Printf("Checking for duplicate emails in range %d-%d", *startIndex, *endIndex)
		if err := utils.CheckForDuplicateEmails(*startIndex, *endIndex); err != nil {
			log.Fatalf("Error checking for duplicate emails: %v", err)
		}

	case "delete":
		deleteCmd.Parse(os.Args[2:])
		log.Printf("Deleting test users in range %d-%d", *deleteStart, *deleteEnd)
		if err := utils.DeleteTestUsers(*deleteStart, *deleteEnd); err != nil {
			log.Fatalf("Error deleting test users: %v", err)
		}

	case "help":
		printHelp()

	default:
		fmt.Printf("Unknown subcommand: %s\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Database utility tool for Diabetify")
	fmt.Println("\nUsage:")
	fmt.Println("  db-tool COMMAND [OPTIONS]")
	fmt.Println("\nCommands:")
	fmt.Println("  seed    Create dummy users for testing")
	fmt.Println("          Options:")
	fmt.Println("            --users=N  Number of dummy users to create (default: 10000)")
	fmt.Println("  check   Check for duplicate emails in the database")
	fmt.Println("          Options:")
	fmt.Println("            --start=N  Start index for email check (default: 0)")
	fmt.Println("            --end=N    End index for email check (default: 1000)")
	fmt.Println("  delete  Delete test users from the database")
	fmt.Println("          Options:")
	fmt.Println("            --start=N  Start index for user deletion (default: 0)")
	fmt.Println("            --end=N    End index for user deletion (default: 1000)")
	fmt.Println("  help    Show this help message")
	fmt.Println("\nEnvironment variables:")
	fmt.Println("  DB_HOST      Database host (default: diabetify-db)")
	fmt.Println("  DB_PORT      Database port (default: 5439)")
	fmt.Println("  DB_USER      Database user (default: postgres)")
	fmt.Println("  DB_PASSWORD  Database password (default: postgres)")
	fmt.Println("  DB_NAME      Database name (default: diabetify)")
	fmt.Println("  DB_SSLMODE   Database SSL mode (default: disable)")
	fmt.Println("  DB_TIMEZONE  Database timezone (default: Asia/Jakarta)")
}
