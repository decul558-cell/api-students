package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

// NewApp merakit aplikasi: membuat instance Fiber, memasang middleware,
// lalu mendaftarkan route. File ini adalah tempat seluruh bagian bertemu.
func NewApp(logger *slog.Logger, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "API Students - Praktikum Backend Lanjut"),
		ErrorHandler: newErrorHandler(logger),
		// Membatasi ukuran body mencegah satu request besar menghabiskan
		// memori server (denial of service yang paling murah dilakukan).
		BodyLimit: 1 * 1024 * 1024, // 1 MB
	})

	middleware.Register(app, logger, GetEnv("ALLOWED_ORIGINS", ""))
	route.Register(app, deps)

	// Penampung terakhir untuk URL yang tidak dikenal.
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}

// newErrorHandler adalah SATU-SATUNYA tempat yang menerjemahkan error
// menjadi response JSON. Handler dan service tidak pernah menulis response
// error secara langsung — mereka cukup me-return *helper.AppError, dan
// Fiber memanggil fungsi ini secara otomatis begitu handler mengembalikan
// error apa pun.
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var appErr *helper.AppError
		if !errors.As(err, &appErr) {
			// Error yang bukan AppError sama sekali (misalnya error bawaan
			// Fiber, atau panic yang ditangkap recover) diperlakukan
			// sebagai kegagalan internal — fail closed, bukan menebak.
			if fe, ok := err.(*fiber.Error); ok {
				appErr = &helper.AppError{Status: fe.Code, Code: "FIBER_ERROR", Message: fe.Message}
			} else {
				appErr = helper.Internal(err)
			}
		}

		requestID, _ := c.Locals("requestid").(string)

		// PERHATIKAN ARAH KONDISI INI: >= 500 berarti kesalahan SISTEM KITA
		// (bug, koneksi database putus, dsb) -> layak Error, perlu ditindak.
		// < 500 berarti kesalahan PEMAKAI API (salah isi, tidak berhak, dsb)
		// -> cukup Warn, bagian normal dari lalu lintas API.
		if appErr.Status >= fiber.StatusInternalServerError {
			logger.Error("request_failed",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.Int("status", appErr.Status),
				slog.String("code", appErr.Code),
				slog.String("error", appErr.Error()),
			)
		} else {
			logger.Warn("request_rejected",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.Int("status", appErr.Status),
				slog.String("code", appErr.Code),
			)
		}

		return c.Status(appErr.Status).JSON(fiber.Map{
			"success":    false,
			"code":       appErr.Code,
			"message":    appErr.Message,
			"fields":     appErr.Fields,
			"request_id": requestID,
		})
	}
}
