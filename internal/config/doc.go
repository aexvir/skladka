// Package config handles the application configuration through environment variables
// and command line flags. It uses github.com/ardanlabs/conf for parsing configuration
// values into strongly typed configuration structs. All environment variables are
// prefixed with "SKD_" automatically.
//
// Configuration can be provided via environment variables or command line flags.
// Command line flags take precedence over environment variables.
//
// Example usage:
//
//	cfg, err := config.Load()
//	if err != nil {
//		if err.Error() == "help requested" {
//			fmt.Println(err)
//			os.Exit(0)
//		}
//		log.Fatal(err)
//	}
//
//	// Access configuration values
//	fmt.Printf("Running in %s environment\n", cfg.Environment)
//	fmt.Printf("Database host: %s:%d\n", cfg.Postgres.Host, cfg.Postgres.Port)
package config
