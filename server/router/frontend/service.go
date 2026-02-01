package frontend

import (
	"context"
	"io/fs"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/internal/util"
	"github.com/hrygo/divinesense/store"
)

type FrontendService struct {
	Profile *profile.Profile
	Store   *store.Store
}

func NewFrontendService(profile *profile.Profile, store *store.Store) *FrontendService {
	return &FrontendService{
		Profile: profile,
		Store:   store,
	}
}

func (*FrontendService) Serve(_ context.Context, e *echo.Echo) {
	// Skipper for Gzip: don't compress API routes (Connect RPC uses binary protobuf)
	gzipSkipper := func(c echo.Context) bool {
		return util.HasPrefixes(c.Path(), "/api", "/memos.api.v1")
	}

	// Add Gzip middleware to compress static assets only
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level:   5,
		Skipper: gzipSkipper,
	}))

	skipper := func(c echo.Context) bool {
		// Skip API routes.
		if util.HasPrefixes(c.Path(), "/api", "/memos.api.v1") {
			return true
		}

		// Security: Prevent MIME type sniffing
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")

		// CORS support for ES modules loaded with crossorigin attribute
		// Vite uses <link rel="modulepreload" crossorigin> which requires CORS headers
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")

		ext := filepath.Ext(c.Path())
		// For index.html, root path, and SPA routes (no extension),
		// set no-cache headers to prevent browser caching.
		// This prevents sensitive data from being accessible via browser back button after logout
		// and ensures users always get the latest version of the application.
		if ext == "" || c.Path() == "/index.html" {
			c.Response().Header().Set(echo.HeaderCacheControl, "no-cache, no-store, must-revalidate")
			c.Response().Header().Set("Pragma", "no-cache")
			c.Response().Header().Set("Expires", "0")
			return false
		}

		// Set Cache-Control header for static assets.
		// Since Vite generates content-hashed filenames (e.g., assets/index-BtVjejZf.js),
		// we can cache aggressively using immutable for files in assets/ directory.
		if util.HasPrefixes(c.Path(), "/assets/") {
			c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
		} else {
			// For other static assets with extensions (like logo.png, favicon.ico), use a shorter max-age
			c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=3600")
		}

		return false
	}

	// Route to serve the main app with HTML5 fallback for SPA behavior.
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: getFileSystem("dist"),
		HTML5:      true, // Enable fallback to index.html
		Skipper:    skipper,
	}))
}

func getFileSystem(path string) http.FileSystem {
	fs, err := fs.Sub(embeddedFiles, path)
	if err != nil {
		panic(err)
	}
	return http.FS(fs)
}
