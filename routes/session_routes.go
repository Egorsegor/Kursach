package routes

import (
	"Dr-Brain-site-project/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func SessionRoutes(app *fiber.App) {
	app.Get("/startquiz/:id", handlers.AuthMiddleware, func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/startquiz.html")
	})
	app.Post("/api/newsession/:id", handlers.CreateSession)
	app.Get("/api/session/:id", handlers.GetSession)
	app.Post("/api/session/:id/join", handlers.JoinSession)
	app.Post("/api/session/:id/leave", handlers.LeaveSession)
	app.Post("/api/session/:id/start", handlers.StartSession)
	app.Post("/api/session/:id/kick", handlers.KickFromSession)

	app.Use("/ws/session/:id", handlers.WebsocketUpgrade)
	app.Get("/ws/session/:id", websocket.New(handlers.SessionWebsocket))

}
