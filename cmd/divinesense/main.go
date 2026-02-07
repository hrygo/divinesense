package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/internal/version"
	"github.com/hrygo/divinesense/server"
	"github.com/hrygo/divinesense/store"
	"github.com/hrygo/divinesense/store/db"
)

var (
	rootCmd = &cobra.Command{
		Use:   "divinesense",
		Short: `An AI-powered personal knowledge assistant. Capture, organize, and retrieve your thoughts with semantic search.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Only load .env for direct binary execution (not when running as systemd service)
			// Systemd service uses /etc/divinesense/config for environment variables
			if !isRunningAsSystemdService() {
				// Try to load .env file from current directory (ignore error if file doesn't exist)
				_ = godotenv.Load() // nolint:errcheck // Error intentionally ignored
			}
			return nil
		},
		Run: func(_ *cobra.Command, _ []string) {
			instanceProfile := &profile.Profile{
				Mode:        viper.GetString("mode"),
				Addr:        viper.GetString("addr"),
				Port:        viper.GetInt("port"),
				UNIXSock:    viper.GetString("unix-sock"),
				Data:        viper.GetString("data"),
				Driver:      viper.GetString("driver"),
				DSN:         viper.GetString("dsn"),
				InstanceURL: viper.GetString("instance-url"),
				Version:     version.GetCurrentVersion(viper.GetString("mode")),
			}
			instanceProfile.FromEnv()
			if err := instanceProfile.Validate(); err != nil {
				panic(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			dbDriver, err := db.NewDBDriver(instanceProfile)
			if err != nil {
				cancel()
				printDatabaseError(err, instanceProfile)
				slog.Error("failed to create db driver", "error", err)
				return
			}

			storeInstance := store.New(dbDriver, instanceProfile)
			if err := storeInstance.Migrate(ctx); err != nil {
				cancel()
				slog.Error("failed to migrate", "error", err)
				return
			}

			s, err := server.NewServer(ctx, instanceProfile, storeInstance)
			if err != nil {
				cancel()
				slog.Error("failed to create server", "error", err)
				return
			}

			c := make(chan os.Signal, 1)
			// Trigger graceful shutdown on SIGINT or SIGTERM.
			// The default signal sent by the `kill` command is SIGTERM,
			// which is taken as the graceful shutdown signal for many systems, eg., Kubernetes, Gunicorn.
			signal.Notify(c, terminationSignals...)

			if err := s.Start(ctx); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					slog.Error("failed to start server", "error", err)
					cancel()
				}
			}

			printGreetings(instanceProfile)

			go func() {
				<-c
				s.Shutdown(ctx)
				cancel()
			}()

			// Wait for CTRL-C.
			<-ctx.Done()
		},
	}
)

func init() {
	viper.SetDefault("mode", "dev")
	viper.SetDefault("driver", "postgres")
	viper.SetDefault("port", 28081)

	// Set version for cobra - use short version for the flag
	rootCmd.Version = version.Version

	// Add version subcommand for detailed version info
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display detailed version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("DivineSense %s\n", version.String())
			fmt.Println(version.StringFull())
		},
	}
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().String("mode", "dev", `mode of server, can be "prod" or "dev" or "demo"`)
	rootCmd.PersistentFlags().String("addr", "", "address of server")
	rootCmd.PersistentFlags().Int("port", 28081, "port of server")
	rootCmd.PersistentFlags().String("unix-sock", "", "path to the unix socket, overrides --addr and --port")
	rootCmd.PersistentFlags().String("data", "", "data directory")
	rootCmd.PersistentFlags().String("driver", "postgres", "database driver (postgres, mysql, sqlite)")
	rootCmd.PersistentFlags().String("dsn", "", "database source name(aka. DSN)")
	rootCmd.PersistentFlags().String("instance-url", "", "the url of your divinesense instance")

	if err := viper.BindPFlag("mode", rootCmd.PersistentFlags().Lookup("mode")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("addr", rootCmd.PersistentFlags().Lookup("addr")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("unix-sock", rootCmd.PersistentFlags().Lookup("unix-sock")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("data", rootCmd.PersistentFlags().Lookup("data")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("driver", rootCmd.PersistentFlags().Lookup("driver")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("dsn", rootCmd.PersistentFlags().Lookup("dsn")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("instance-url", rootCmd.PersistentFlags().Lookup("instance-url")); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("memos")
	viper.AutomaticEnv()
	// Support both DIVINESENSE_* and MEMOS_* prefixes for backward compatibility
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Bind environment variables for configuration
	// Try DIVINESENSE_* first, fall back to MEMOS_*
	bindEnvWithFallback := func(configKey, newEnv, legacyEnv string) {
		if err := viper.BindEnv(configKey, newEnv); err != nil {
			panic(err)
		}
		// Also bind legacy prefix for compatibility
		if err := viper.BindEnv(configKey, legacyEnv); err != nil {
			panic(err)
		}
	}

	bindEnvWithFallback("driver", "DIVINESENSE_DRIVER", "MEMOS_DRIVER")
	bindEnvWithFallback("dsn", "DIVINESENSE_DSN", "MEMOS_DSN")
	bindEnvWithFallback("instance-url", "DIVINESENSE_INSTANCE_URL", "MEMOS_INSTANCE_URL")
}

func printGreetings(profile *profile.Profile) {
	fmt.Printf("DivineSense %s started successfully!\n", profile.Version)

	if profile.IsDev() {
		fmt.Fprint(os.Stderr, "Development mode is enabled\n")
		if profile.DSN != "" {
			fmt.Fprintf(os.Stderr, "Database: %s\n", profile.DSN)
		}
	}

	// Server information
	fmt.Printf("Data directory: %s\n", profile.Data)
	fmt.Printf("Database driver: %s\n", profile.Driver)
	fmt.Printf("Mode: %s\n", profile.Mode)

	// Connection information
	if len(profile.UNIXSock) == 0 {
		if len(profile.Addr) == 0 {
			fmt.Printf("Server running on port %d\n", profile.Port)
			fmt.Printf("Access DivineSense at: http://localhost:%d\n", profile.Port)
		} else {
			fmt.Printf("Server running on %s:%d\n", profile.Addr, profile.Port)
			fmt.Printf("Access DivineSense at: http://%s:%d\n", profile.Addr, profile.Port)
		}
	} else {
		fmt.Printf("Server running on unix socket: %s\n", profile.UNIXSock)
	}

	fmt.Println()
	fmt.Printf("Documentation: %s\n", "https://github.com/hrygo/divinesense")
	fmt.Printf("Source code: %s\n", "https://github.com/hrygo/divinesense")
	fmt.Println("\nHappy note-taking!")
}

// isRunningAsSystemdService detects if the process is running under systemd
func isRunningAsSystemdService() bool {
	// Check if invoked by systemd (environment variables set by systemd)
	return os.Getenv("INVOCATION_ID") != "" || os.Getenv("WATCHDOG_USEC") != ""
}

// printDatabaseError provides user-friendly error messages for database connection issues
func printDatabaseError(err error, profile *profile.Profile) {
	fmt.Fprintln(os.Stderr, "\n❌ Database Connection Failed")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "cannot connect") || strings.Contains(errMsg, "localhost:25432"):
		fmt.Fprintln(os.Stderr, "\n📌 PostgreSQL is not running.")
		fmt.Fprintf(os.Stderr, "\n   Start PostgreSQL with:\n")
		if profile.Driver == "postgres" {
			fmt.Fprintf(os.Stderr, "   ■ Docker:  docker compose -f docker/compose/dev.yml up -d postgres\n")
			fmt.Fprintf(os.Stderr, "   ■ System:  sudo systemctl start postgresql\n")
		}
		fmt.Fprintf(os.Stderr, "\n   Or use SQLite for development (no AI features):\n")
		fmt.Fprintf(os.Stderr, "   ■ Set: DIVINESENSE_DRIVER=sqlite\n")
		fmt.Fprintf(os.Stderr, "   ■ Or:   ./divinesense --driver=sqlite --data=./data\n")

	case strings.Contains(errMsg, "SSL is not enabled") || strings.Contains(errMsg, "sslmode"):
		fmt.Fprintln(os.Stderr, "\n📌 PostgreSQL SSL configuration mismatch.")
		fmt.Fprintf(os.Stderr, "\n   Add ?sslmode=disable to your DSN:\n")
		fmt.Fprintf(os.Stderr, "   ■ export DIVINESENSE_DSN=\"postgres://user:pass@localhost:25432/dbname?sslmode=disable\"\n")

	case strings.Contains(errMsg, "password authentication failed") || strings.Contains(errMsg, "auth"):
		fmt.Fprintln(os.Stderr, "\n📌 PostgreSQL authentication failed.")
		fmt.Fprintf(os.Stderr, "\n   Check your credentials in the DSN or .env file.\n")

	case strings.Contains(errMsg, "database") && strings.Contains(errMsg, "does not exist"):
		fmt.Fprintln(os.Stderr, "\n📌 Database does not exist.")
		fmt.Fprintf(os.Stderr, "\n   Create it with:\n")
		fmt.Fprintf(os.Stderr, "   ■ docker exec -it postgres psql -U postgres -c \"CREATE DATABASE divinesense;\"\n")

	case strings.Contains(errMsg, "permission denied"):
		fmt.Fprintln(os.Stderr, "\n📌 Permission denied.")
		fmt.Fprintf(os.Stderr, "\n   Check database user permissions.\n")
		if strings.Contains(errMsg, "schema") {
			fmt.Fprintf(os.Stderr, "   ■ Run: GRANT ALL ON SCHEMA public TO divinesense;\n")
		}

	default:
		fmt.Fprintln(os.Stderr, "\n📌 Error:", errMsg)
	}

	// Check if .env file exists
	if _, statErr := os.Stat(".env"); statErr == nil {
		fmt.Fprintf(os.Stderr, "\n💡 Found .env file - configuration loaded from current directory.\n")
	} else {
		fmt.Fprintf(os.Stderr, "\n💡 Tip: Create a .env file for local configuration (see .env.example)\n")
	}

	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
