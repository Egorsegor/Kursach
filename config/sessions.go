package config

import (
	"os"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/mongodb"
)

var Store *session.Store

func ConnectSessions() {
	Storage := mongodb.New(mongodb.Config{
		ConnectionURI: os.Getenv("MONGODB_URI"),
		Database:      "dr_brain_db",
		Collection:    "tokens",
	})

	Store = session.New(session.Config{
		Storage: Storage,
	})
}
