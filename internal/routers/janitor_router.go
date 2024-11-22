package routers

import (
	"Boxed/internal/cmd/janitor"
	"github.com/gofiber/fiber/v2"
)

func SetupJanitorRouter(app *fiber.App, janitor *janitor.Janitor) {
	app.Post("/janitor/clean", func(ctx *fiber.Ctx) error {
		err := janitor.ForceStartCleanCycle()
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{})
	})
}
