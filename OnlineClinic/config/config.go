package config

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// Define global variables for the database connection and configuration.
var (
	DB  *sql.DB // Global database connection object
	Cfg Config  // Global configuration object
)

// Define a struct to hold configuration values for the application.
type Config struct {
	DBUser     string // Database username
	DBPassword string // Database password
	DBHost     string // Database host (e.g., IP address or hostname)
	DBPort     string // Database port (e.g., 3306 for MySQL)
	DBName     string // Name of the database to connect to
	JWTSecret  string // Secret key used for JSON Web Token (JWT) signing
}

// LoadConfig initializes the application configuration.
func LoadConfig() {
	Cfg = Config{
		DBUser:     "root",
		DBPassword: "gApczcyLDZFavkfufpe7iA6t",
		DBHost:     "damavand.liara.cloud",
		DBPort:     "33525",
		DBName:     "OnlineClinic",
		JWTSecret:  "superS3cr3tK3y!123#MyClinicApp",
	}

	log.Printf("Loaded configuration: %+v\n", Cfg)
	log.Println("Configuration loaded successfully.")

	ConnectDB()
	log.Println("Database connection established successfully.")
}

// ConnectDB establishes a connection to the MySQL database.
func ConnectDB() {
	var err error

	// Connection string without TLS
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		Cfg.DBUser, Cfg.DBPassword, Cfg.DBHost, Cfg.DBPort, Cfg.DBName)

	// Open the database connection
	DB, err = sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("Error connecting to the %s database: %v", Cfg.DBName, err)
	}

	// Test the connection
	err = DB.Ping()
	if err != nil {
		// log.Fatalf("Error connecting to the database: %v", err)
	}

	fmt.Printf("Successfully connected to the %s database\n", Cfg.DBName)
}
