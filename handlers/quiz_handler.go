package handlers

import (
	"Dr-Brain-site-project/config"
	"Dr-Brain-site-project/models"
	"context"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateQuiz(c *fiber.Ctx) error {
	var quiz models.Quiz

	if err := c.BodyParser(&quiz); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат данных",
		})
	}

	for i := range quiz.Questions {
		InsertResult, err := config.QuestionsCollection.InsertOne(context.Background(), quiz.Questions[i])
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Ошибка генерации ID вопроса"})
		}
		quiz.Questions[i].ID = InsertResult.InsertedID.(primitive.ObjectID)
	}

	InsertResult, err := config.QuizCollection.InsertOne(context.Background(), quiz)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Не удалось сохранить квиз"})
	}

	quiz.ID = InsertResult.InsertedID.(primitive.ObjectID)

	return c.Status(200).JSON(quiz)
}

func GetQuizByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	quizID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Wrong ID"})
	}

	var quiz models.Quiz
	err = config.QuizCollection.FindOne(context.Background(), bson.M{"_id": quizID}).Decode(&quiz)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Quiz is not found"})
	}

	return c.JSON(quiz)
}

func GetQuizzes(c *fiber.Ctx) error {
	var quizzes []models.Quiz

	cursor, err := config.QuizCollection.Find(context.Background(), bson.M{})

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var quiz models.Quiz
		if err := cursor.Decode(&quiz); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		quizzes = append(quizzes, quiz)
	}

	return c.JSON(quizzes)
}

func GetQuestionByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	questionID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Wrong ID"})
	}

	var question models.Question
	err = config.QuestionsCollection.FindOne(context.Background(), bson.M{"_id": questionID}).Decode(&question)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Question is not found"})
	}

	return c.JSON(question)
}

func CheckAnswers(c *fiber.Ctx) error {
	type request struct {
		Answer []int `json:"answer"`
	}

	var req request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	questionID := c.Params("id")
	var question models.Question
	err := config.QuestionsCollection.FindOne(context.Background(), bson.M{"_id": questionID}).Decode(&question)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Question is not found"})
	}

	isCorrect := models.CompareAnswers(question.CorrectAnswer, req.Answer)

	return c.JSON(fiber.Map{"correct": isCorrect})
}

func GetQuizzesByUserID(c *fiber.Ctx) error {
	idParam := c.Params("userid")
	quizID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Wrong ID"})
	}

	var quizzes []models.Quiz
	cursor, err := config.QuizCollection.Find(context.Background(), bson.M{"userid": quizID})
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Quiz is not found"})
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &quizzes); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to decode quizzes"})
	}

	return c.JSON(quizzes)
}
