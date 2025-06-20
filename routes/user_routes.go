package routes

import (
	"Dr-Brain-site-project/handlers"

	"github.com/gofiber/fiber/v2"
)

func UserRoutes(app *fiber.App) {
	app.Get("/admin/users", handlers.GetUsers)
	app.Post("api/registration", handlers.NewUser)
	app.Patch("/emailverified/:id", handlers.EmailVerification)
	app.Delete("/deleteuser/:id", handlers.DeleteUser)
	app.Post("api/login", handlers.LoginUser)
	app.Get("/login", func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/login.html")
	})
	app.Get("/registration", func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/register.html")
	})
}
