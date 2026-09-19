package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"api-students/helper"
)

// Register memasang seluruh middleware yang berlaku untuk semua route.
// URUTAN PENTING: middleware dieksekusi sesuai urutan pemasangan.
func Register(app *fiber.App, logger *slog.Logger, allowedOrigins string) {
	app.Use(requestid.New())            // 1. beri setiap request satu ID unik
	app.Use(recover.New())              // 2. tangkap panic agar server tidak mati
	app.Use(helmet.New())               // 3. pasang header keamanan dasar
	app.Use(corsPolicy(allowedOrigins)) // 4. BERUBAH: tidak lagi cors.New() polos
	app.Use(RequestLogger(logger))      // 5. catat setiap request
}

// corsPolicy membatasi origin yang boleh memanggil API.
// cors.New() tanpa konfigurasi mengizinkan SEMUA origin — cukup untuk
// latihan sebelumnya, tetapi tidak untuk API yang memakai token.
func corsPolicy(allowedOrigins string) fiber.Handler {
	if strings.TrimSpace(allowedOrigins) == "" {
		allowedOrigins = "http://localhost:5173"
	}
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	})
}

// RequestLogger mencatat setiap request ke log terstruktur.
// Bila request sudah melewati RequireAuth, user_id dan role pemanggil
// ikut dicatat — berguna untuk audit siapa melakukan apa, terutama
// pada endpoint yang diatur RBAC seperti students.
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		requestID, _ := c.Locals("requestid").(string)

		fields := []any{
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		}

		if authUser, ok := helper.CurrentUser(c); ok {
			fields = append(fields,
				slog.Int("user_id", authUser.UserID),
				slog.String("role", authUser.Role),
			)
		}

		logger.Info("http_request", fields...)
		return err
	}
}

var methodsWithBody = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

// RequireJSON menolak request berisi body yang Content-Type-nya bukan JSON.
func RequireJSON(c *fiber.Ctx) error {
	if methodsWithBody[c.Method()] {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return helper.Fail(c, fiber.StatusUnsupportedMediaType,
				"Content-Type harus application/json")
		}
	}
	return c.Next()
}
