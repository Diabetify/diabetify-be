package main

import (
	"diabetify/database"
	"diabetify/internal/utils"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file from project root
	if err := godotenv.Load(); err != nil {
		// Try loading from parent directory (in case running from cmd/seed/)
		if err := godotenv.Load("../../.env"); err != nil {
			log.Printf("Warning: No .env file found: %v", err)
		}
	}
}

func main() {
	seedCmd := flag.NewFlagSet("seed", flag.ExitOnError)
	numUsers := seedCmd.Int("users", utils.DefaultNumUsers, "Number of dummy users to create")
	startID := seedCmd.Int("start-id", 0, "Starting user ID (0 means auto-increment)")
	endID := seedCmd.Int("end-id", 0, "Ending user ID (0 means use --users count)")
	shardName := seedCmd.String("shard", "", "Target specific shard (shard1 or shard2)")
	useSharding := seedCmd.Bool("sharded", false, "Use sharded database (default: false for backward compatibility)")

	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
	startIndex := checkCmd.Int("start", 0, "Start index for email check")
	endIndex := checkCmd.Int("end", 1000, "End index for email check")
	checkSharded := checkCmd.Bool("sharded", false, "Check sharded database (default: false)")

	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteStart := deleteCmd.Int("start", 0, "Start index for user deletion")
	deleteEnd := deleteCmd.Int("end", 1000, "End index for user deletion")
	deleteSharded := deleteCmd.Bool("sharded", false, "Delete from sharded database (default: false)")
	deleteShard := deleteCmd.String("shard", "", "Delete from specific shard (shard1, shard2, or all)")

	clearCmd := flag.NewFlagSet("clear", flag.ExitOnError)
	clearShard := clearCmd.String("shard", "all", "Clear specific shard (shard1, shard2, or all)")

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	// Debug: Print environment variables
	log.Printf("DB_HOST: %s", os.Getenv("DB_HOST"))
	log.Printf("DB_HOST2: %s", os.Getenv("DB_HOST2"))
	log.Printf("DB_PORT: %s", os.Getenv("DB_PORT"))
	log.Printf("DB_USER: %s", os.Getenv("DB_USER"))
	log.Printf("DB_NAME: %s", os.Getenv("DB_NAME"))
	log.Printf("DB_SSLMODE: %s", os.Getenv("DB_SSLMODE"))

	// Parse the subcommand
	switch os.Args[1] {
	case "seed":
		seedCmd.Parse(os.Args[2:])

		// Initialize database connection
		if *useSharding {
			log.Println("🔄 Connecting to sharded database...")
			database.ConnectShardedDatabase()
		}

		// Handle ID range seeding
		if *startID > 0 && *endID > 0 {
			if *endID <= *startID {
				log.Fatalf("End ID (%d) must be greater than start ID (%d)", *endID, *startID)
			}
			usersToCreate := *endID - *startID + 1
			log.Printf("Starting user seeder for ID range %d-%d (%d users)", *startID, *endID, usersToCreate)

			if *useSharding {
				if err := utils.SeedUsersWithIDRangeSharded(*startID, *endID, *shardName); err != nil {
					log.Fatalf("Error seeding users with ID range (sharded): %v", err)
				}
			} else {
				if err := utils.SeedUsersWithIDRange(*startID, *endID); err != nil {
					log.Fatalf("Error seeding users with ID range: %v", err)
				}
			}
		} else {
			log.Printf("Starting user seeder with %d users", *numUsers)

			if *useSharding {
				if err := utils.SeedUsersSharded(*numUsers, *shardName); err != nil {
					log.Fatalf("Error seeding users (sharded): %v", err)
				}
			} else {
				if err := utils.SeedUsers(*numUsers); err != nil {
					log.Fatalf("Error seeding users: %v", err)
				}
			}
		}

	case "check":
		checkCmd.Parse(os.Args[2:])

		if *checkSharded {
			database.ConnectShardedDatabase()
		}

		log.Printf("Checking for duplicate emails in range %d-%d", *startIndex, *endIndex)

		if *checkSharded {
			if err := utils.CheckForDuplicateEmailsSharded(*startIndex, *endIndex); err != nil {
				log.Fatalf("Error checking for duplicate emails (sharded): %v", err)
			}
		} else {
			if err := utils.CheckForDuplicateEmails(*startIndex, *endIndex); err != nil {
				log.Fatalf("Error checking for duplicate emails: %v", err)
			}
		}

	case "delete":
		deleteCmd.Parse(os.Args[2:])

		if *deleteSharded {
			database.ConnectShardedDatabase()
		}

		log.Printf("Deleting test users in range %d-%d", *deleteStart, *deleteEnd)

		if *deleteSharded {
			if err := utils.DeleteTestUsersSharded(*deleteStart, *deleteEnd, *deleteShard); err != nil {
				log.Fatalf("Error deleting test users (sharded): %v", err)
			}
		} else {
			if err := utils.DeleteTestUsers(*deleteStart, *deleteEnd); err != nil {
				log.Fatalf("Error deleting test users: %v", err)
			}
		}

	case "clear":
		clearCmd.Parse(os.Args[2:])
		database.ConnectShardedDatabase()

		log.Printf("Clearing all data from shard: %s", *clearShard)
		if err := utils.ClearAllDataSharded(*clearShard); err != nil {
			log.Fatalf("Error clearing data (sharded): %v", err)
		}

	case "setup-shards":
		log.Println("🚀 Setting up sharded database with sample data...")
		database.ConnectShardedDatabase()

		// Clear all existing data
		log.Println("🗑️  Clearing all existing data...")
		if err := utils.ClearAllDataSharded("all"); err != nil {
			log.Fatalf("Error clearing data: %v", err)
		}

		// Seed shard1 with users 1-5000
		log.Println("📊 Seeding shard1 with users 1-5000...")
		if err := utils.SeedUsersWithIDRangeSharded(1, 5000, "shard1"); err != nil {
			log.Fatalf("Error seeding shard1: %v", err)
		}

		// Seed shard2 with users 5001-10000
		log.Println("📊 Seeding shard2 with users 5001-10000...")
		if err := utils.SeedUsersWithIDRangeSharded(5001, 10000, "shard2"); err != nil {
			log.Fatalf("Error seeding shard2: %v", err)
		}

		// Show final stats
		if counts, err := utils.GetUserCountSharded(); err == nil {
			log.Println("✅ Sharded database setup complete!")
			for shard, count := range counts {
				if shard != "total" {
					log.Printf("   - %s: %d users", shard, count)
				}
			}
			log.Printf("   - Total: %d users", counts["total"])
		}
	case "seed-proper":
		seedProperCmd := flag.NewFlagSet("seed-proper", flag.ExitOnError)
		totalUsers := seedProperCmd.Int("users", 10000, "Total number of users to create with global unique IDs")

		seedProperCmd.Parse(os.Args[2:])

		log.Printf("🌱 Seeding %d users with proper global unique IDs across shards...", *totalUsers)
		database.ConnectShardedDatabase()

		if err := utils.SeedUsersShardedWithGlobalIDs(*totalUsers); err != nil {
			log.Fatalf("Error seeding users with global IDs: %v", err)
		}

		log.Println("✅ Seeding completed! Generating reports...")

		// Show distribution report
		if err := utils.GetUserDistributionReport(); err != nil {
			log.Printf("Warning: Could not generate distribution report: %v", err)
		}

		// Verify sharding consistency
		if err := utils.VerifyShardingConsistency(); err != nil {
			log.Printf("Warning: Could not verify consistency: %v", err)
		}

		// Show final stats
		if counts, err := utils.GetUserCountSharded(); err == nil {
			log.Println("📊 Final Statistics:")
			for shard, count := range counts {
				log.Printf("   %s: %d users", shard, count)
			}
		}
	case "stats":
		// New command to show database statistics
		database.ConnectShardedDatabase()
		log.Println("📊 Database Statistics:")

		if counts, err := utils.GetUserCountSharded(); err == nil {
			for shard, count := range counts {
				log.Printf("   %s: %d users", shard, count)
			}
		} else {
			log.Fatalf("Error getting stats: %v", err)
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
	fmt.Println("Database utility tool for Diabetify (Now with Sharding Support!)")
	fmt.Println("\nUsage:")
	fmt.Println("  db-tool COMMAND [OPTIONS]")
	fmt.Println("\nCommands:")
	fmt.Println("  seed         Create dummy users for testing")
	fmt.Println("               Options:")
	fmt.Println("                 --users=N       Number of dummy users to create (default: 10000)")
	fmt.Println("                 --start-id=N    Starting user ID (0 means auto-increment)")
	fmt.Println("                 --end-id=N      Ending user ID (0 means use --users count)")
	fmt.Println("                 --shard=NAME    Target specific shard (shard1 or shard2)")
	fmt.Println("                 --sharded=BOOL  Use sharded database (default: false)")
	fmt.Println("")
	fmt.Println("  check        Check for duplicate emails in the database")
	fmt.Println("               Options:")
	fmt.Println("                 --start=N       Start index for email check (default: 0)")
	fmt.Println("                 --end=N         End index for email check (default: 1000)")
	fmt.Println("                 --sharded=BOOL  Check sharded database (default: false)")
	fmt.Println("")
	fmt.Println("  delete       Delete test users from the database")
	fmt.Println("               Options:")
	fmt.Println("                 --start=N       Start index for user deletion (default: 0)")
	fmt.Println("                 --end=N         End index for user deletion (default: 1000)")
	fmt.Println("                 --shard=NAME    Delete from specific shard (shard1, shard2, or all)")
	fmt.Println("                 --sharded=BOOL  Delete from sharded database (default: false)")
	fmt.Println("")
	fmt.Println("  clear        Clear all data from database/shards")
	fmt.Println("               Options:")
	fmt.Println("                 --shard=NAME    Clear specific shard (shard1, shard2, or all) (default: all)")
	fmt.Println("")
	fmt.Println("  setup-shards One-command setup: Clear all data and seed both shards properly")
	fmt.Println("               (shard1: users 1-5000, shard2: users 5001-10000)")
	fmt.Println("")
	fmt.Println("  stats        Show database statistics across all shards")
	fmt.Println("")
	fmt.Println("  help         Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Traditional single database")
	fmt.Println("  db-tool seed --start-id=2002 --end-id=2009          # Seed users with IDs 2002-2009")
	fmt.Println("  db-tool seed --users=1000                           # Seed 1000 users with auto-increment IDs")
	fmt.Println("")
	fmt.Println("  # Sharded database")
	fmt.Println("  db-tool setup-shards                                # Quick setup: Clear all and seed both shards")
	fmt.Println("  db-tool seed --start-id=1 --end-id=5000 --shard=shard1 --sharded=true")
	fmt.Println("  db-tool seed --start-id=5001 --end-id=10000 --shard=shard2 --sharded=true")
	fmt.Println("  db-tool stats                                       # Show user count per shard")
	fmt.Println("  db-tool clear --shard=shard1                       # Clear specific shard")
	fmt.Println("  db-tool delete --start=1 --end=5000 --shard=shard1 --sharded=true")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  DB_HOST      Database host (default: diabetify-db)")
	fmt.Println("  DB_HOST2     Second shard host (for sharding)")
	fmt.Println("  DB_PORT      Database port (default: 5439)")
	fmt.Println("  DB_USER      Database user (default: postgres)")
	fmt.Println("  DB_PASSWORD  Database password (default: postgres)")
	fmt.Println("  DB_NAME      Database name (default: diabetify)")
	fmt.Println("  DB_SSLMODE   Database SSL mode (default: require)")
	fmt.Println("  DB_TIMEZONE  Database timezone (default: Asia/Jakarta)")
}
