package routes

import (
	"Dr-Brain-site-project/handlers"

	"github.com/gofiber/fiber/v2"
)

func QuizRoutes(app *fiber.App) {
	app.Get("/newquiz", handlers.AuthMiddleware, func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/newquiz.html")
	})
	app.Post("/api/newquiz", handlers.CreateQuiz)
	app.Get("/quiz/:id", handlers.AuthMiddleware, func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/quiz.html")
	})
	app.Get("/api/quiz/:id", handlers.AuthMiddleware, handlers.GetQuizByID)
	app.Get("/api/question/:id", handlers.GetQuestionByID)
	app.Post("/question/:id/check", handlers.AuthMiddleware, handlers.SubmitAnswer)
	app.Post("/session/:id/finish", handlers.AuthMiddleware, handlers.FinishQuiz)
	app.Get("/admin/quizzes", handlers.GetQuizzes)
	app.Get("/dashboard", handlers.AuthMiddleware, func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/dashboard.html")
	})
	app.Get("/dashboard/:userid", handlers.GetQuizzesByUserID)
	app.Get("/quiz/session/:id", handlers.AuthMiddleware, func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/runningquiz.html")
	})
}
