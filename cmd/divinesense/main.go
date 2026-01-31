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

func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
