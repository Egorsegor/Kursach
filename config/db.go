package config

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database
var Client *mongo.Client
var UserCollection *mongo.Collection
var QuizCollection *mongo.Collection
var QuestionsCollection *mongo.Collection
var SessionCollection *mongo.Collection
var TokenCollection *mongo.Collection

func ConnectDB() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Couldn't load .env file", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	MONGODB_URI := os.Getenv("MONGODB_URI")
	clientOptions := options.Client().ApplyURI(MONGODB_URI)
	Client, err = mongo.Connect(ctx, clientOptions)

	if err != nil {
		log.Fatal(err)
	}

	if err := Client.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully connected to MongoDB")

	DB = Client.Database("dr_brain_db")

	UserCollection = DB.Collection("users")
	QuizCollection = DB.Collection("quizes")
	QuestionsCollection = DB.Collection("questions")
	SessionCollection = DB.Collection("sessions")
	TokenCollection = DB.Collection("tokens")

	ensureIndexes(ctx, UserCollection)
}

func ensureIndexes(ctx context.Context, collection *mongo.Collection) {
	emailIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	loginIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "login", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateOne(ctx, emailIndexModel)
	if err != nil {
		log.Fatal("Error creating email indexes:", err.Error())
	}
	log.Println("Email indexes are created successfully")

	_, err = collection.Indexes().CreateOne(ctx, loginIndexModel)
	if err != nil {
		log.Fatal("Error creating login indexes:", err.Error())
	}
	log.Println("Login indexes are created successfully")
}
