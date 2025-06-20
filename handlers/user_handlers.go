package handlers

import (
	"Dr-Brain-site-project/config"
	"Dr-Brain-site-project/models"
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

func GetUsers(c *fiber.Ctx) error {
	var users []models.User

	cursor, err := config.UserCollection.Find(context.Background(), bson.M{})

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		users = append(users, user)
	}

	return c.JSON(users)
}

func NewUser(c *fiber.Ctx) error {
	tempUser := new(models.TempUser)

	if err := c.BodyParser(tempUser); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	if tempUser.Password != tempUser.PasswordConfirm {
		return c.Status(400).JSON(fiber.Map{"error": "Passwords should match!"})
	}

	if tempUser.Password == "" || tempUser.Login == "" || tempUser.Email == "" || tempUser.PasswordConfirm == "" {
		return c.Status(400).JSON(fiber.Map{"error": "All fields must be filled in!"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempUser.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Ошибка при хэшировании пароля"})
	}

	user := models.User{
		Login:         tempUser.Login,
		Email:         tempUser.Email,
		EmailVerified: false,
		Password:      string(hashedPassword),
	}

	InsertResult, err := config.UserCollection.InsertOne(context.Background(), user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			if strings.Contains(err.Error(), "email_1") {
				return c.Status(409).JSON(fiber.Map{"error": "This Email is already taken"})
			} else if strings.Contains(err.Error(), "login_1") {
				return c.Status(409).JSON(fiber.Map{"error": "This Login is already taken"})
			}
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	user.ID = InsertResult.InsertedID.(primitive.ObjectID)

	sess, err := config.Store.Get(c)
	if err != nil {
		return err
	}
	sess.Set("userID", user.ID.Hex())
	if err := sess.Save(); err != nil {
		return err
	}

	return c.Status(201).JSON(user)
}

func EmailVerification(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	filter := bson.M{"_id": userID}
	update := bson.M{"$set": bson.M{"emailverified": true}}

	_, err = config.UserCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"success": true})
}

func DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	ObjectID, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid User ID"})
	}

	filter := bson.M{"_id": ObjectID}
	_, err = config.UserCollection.DeleteOne(context.Background(), filter)

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"success": true})
}

func LoginUser(c *fiber.Ctx) error {
	type LoginRequest struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}

	var req LoginRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	filter := bson.M{
		"$or": []bson.M{
			{"email": req.Identifier},
			{"login": req.Identifier},
		},
	}

	var user models.User
	err := config.UserCollection.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	sess, err := config.Store.Get(c)
	if err != nil {
		return err
	}
	sess.Set("userID", user.ID.Hex())
	if err := sess.Save(); err != nil {
		return err
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "Login successful",
		"user": fiber.Map{
			"id":            user.ID.Hex(),
			"login":         user.Login,
			"email":         user.Email,
			"emailVerified": user.EmailVerified,
		},
	})
}

func AuthMiddleware(c *fiber.Ctx) error {
	sess, err := config.Store.Get(c)
	if err != nil {
		return err
	}

	userID := sess.Get("userID")
	if userID == nil {
		return c.Redirect("/login")
	}

	c.Locals("userID", userID)
	return c.Next()
}
