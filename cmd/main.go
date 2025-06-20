package main

import (
	"Dr-Brain-site-project/config"
	"Dr-Brain-site-project/routes"
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Couldn't load .env file", err)
	}

	config.ConnectDB()
	defer config.Client.Disconnect(context.Background())

	config.ConnectSessions()

	app := fiber.New()

	app.Use(cors.New())

	app.Static("/", "./frontend")

	routes.UserRoutes(app)
	routes.QuizRoutes(app)
	routes.SessionRoutes(app)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}
