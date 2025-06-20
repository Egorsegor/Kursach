package handlers

import (
	"Dr-Brain-site-project/config"
	"Dr-Brain-site-project/models"
	"context"
	"log"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var sessions = make(map[primitive.ObjectID]*models.Session)
var sessionConnections = make(map[primitive.ObjectID][]*websocket.Conn)
var connectedUsers = map[primitive.ObjectID]map[primitive.ObjectID]bool{}

type RunningQuiz struct {
	QuizID      primitive.ObjectID
	QuestionIdx int
	StartTime   time.Time
	Answers     map[primitive.ObjectID][]bool
	Answered    map[primitive.ObjectID]bool
	Finished    map[primitive.ObjectID]bool
	Current     *models.Question
}

var runningQuizzes = map[primitive.ObjectID]*RunningQuiz{}

func compareAnswers(correct, submitted []int) bool {
	if len(correct) != len(submitted) {
		return false
	}
	m := map[int]bool{}
	for _, ans := range correct {
		m[ans] = true
	}
	for _, sub := range submitted {
		if !m[sub] {
			return false
		}
	}
	return true
}

func getQuizByID(id primitive.ObjectID) models.Quiz {
	var quiz models.Quiz
	err := config.QuizCollection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&quiz)
	if err != nil {
		log.Println("quiz not found", err)
	}
	return quiz
}

func startQuestionTimer(sessionID primitive.ObjectID, quiz models.Quiz) {
	rq := runningQuizzes[sessionID]

	for {
		if rq.QuestionIdx >= len(quiz.Questions) {
			broadcastEnd(sessionID)
			return
		}

		q := quiz.Questions[rq.QuestionIdx]
		rq.StartTime = time.Now()
		broadcastQuestion(sessionID, q)

		select {
		case <-time.After(60 * time.Second):
			for uid := range connectedUsers[sessionID] {
				if !rq.Answered[uid] {
					rq.Answers[uid] = append(rq.Answers[uid], false)
				}
				rq.Answered[uid] = false
			}
			rq.QuestionIdx++

		case <-func() chan struct{} {
			ch := make(chan struct{})
			go func() {
				for {
					time.Sleep(1 * time.Second)
					if len(rq.Finished) == len(connectedUsers[sessionID]) {
						ch <- struct{}{}
						return
					}
				}
			}()
			return ch
		}():
			sendRating(sessionID)
			return
		}
	}
}

func broadcastQuestion(sessionID primitive.ObjectID, q models.Question) {
	if rq, ok := runningQuizzes[sessionID]; ok {
		rq.Current = &q
	}

	msg := fiber.Map{
		"type": "question",
		"question": fiber.Map{
			"_id":     q.ID.Hex(),
			"text":    q.Text,
			"type":    q.Type,
			"options": q.Options,
		},
		"timeout": 60,
	}

	for _, conn := range sessionConnections[sessionID] {
		conn.WriteJSON(msg)
	}
}

func broadcastEnd(sessionID primitive.ObjectID) {
	for _, conn := range sessionConnections[sessionID] {
		conn.WriteJSON(fiber.Map{"type": "waiting"})
	}
	sendRating(sessionID)
}

func sendRating(sessionID primitive.ObjectID) {
	rq, ok := runningQuizzes[sessionID]
	if !ok {
		log.Printf("sendRating: no running quiz for session %v", sessionID.Hex())
		return
	}

	type userScore struct {
		UserID string `json:"userID"`
		Login  string `json:"login"`
		Score  int    `json:"score"`
	}

	results := []userScore{}
	for uid, arr := range rq.Answers {
		var user models.User
		err := config.UserCollection.FindOne(context.Background(), bson.M{"_id": uid}).Decode(&user)
		login := "unknown"
		if err == nil {
			login = user.Login
		} else {
			log.Printf("sendRating: failed to get login for user %v: %v", uid.Hex(), err)
		}

		score := 0
		for _, correct := range arr {
			if correct {
				score++
			}
		}

		results = append(results, userScore{
			UserID: uid.Hex(),
			Login:  login,
			Score:  score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	for _, conn := range sessionConnections[sessionID] {
		conn.WriteJSON(fiber.Map{
			"type":   "rating",
			"rating": results,
		})
	}

	delete(runningQuizzes, sessionID)
}

func SubmitAnswer(c *fiber.Ctx) error {
	var body struct {
		SessionID string `json:"session_id"`
		Answers   []int  `json:"answers"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	sid, err := primitive.ObjectIDFromHex(body.SessionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid session ID"})
	}

	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
	}

	rq, ok := runningQuizzes[sid]
	if !ok || rq == nil {
		log.Printf("SubmitAnswer: no running quiz for session %s", sid.Hex())
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "quiz not running"})
	}

	quiz := getQuizByID(rq.QuizID)
	if quiz.ID.IsZero() || rq.QuestionIdx >= len(quiz.Questions) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "invalid quiz data"})
	}

	q := quiz.Questions[rq.QuestionIdx]
	correct := compareAnswers(q.CorrectAnswer, body.Answers)
	rq.Answers[userID] = append(rq.Answers[userID], correct)
	rq.Answered[userID] = true

	allAnswered := true
	for uid := range connectedUsers[sid] {
		if !rq.Answered[uid] {
			allAnswered = false
			break
		}
	}

	if allAnswered {
		rq.QuestionIdx++
		if rq.QuestionIdx < len(quiz.Questions) {
			rq.StartTime = time.Now()
			for uid := range connectedUsers[sid] {
				rq.Answered[uid] = false
			}
			broadcastQuestion(sid, quiz.Questions[rq.QuestionIdx])
		} else {
			broadcastEnd(sid)
		}
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func FinishQuiz(c *fiber.Ctx) error {
	sid, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid session id"})
	}
	userIDStr := c.Locals("userID").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user id"})
	}

	rq, ok := runningQuizzes[sid]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "quiz not running"})
	}

	rq.Finished[userID] = true

	if len(rq.Finished) == len(connectedUsers[sid]) {
		sendRating(sid)
	}

	return c.JSON(fiber.Map{"status": "waiting"})
}

func CreateSession(c *fiber.Ctx) error {
	quizID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid quiz ID"})
	}

	sess, err := config.Store.Get(c)
	if err != nil {
		return err
	}
	idStr := sess.Get("userID").(string)
	userID, _ := primitive.ObjectIDFromHex(idStr)

	session := models.Session{
		QuizID:       quizID,
		CreatorID:    userID,
		Participants: []primitive.ObjectID{userID},
		IsActive:     true,
		Started:      false,
	}

	result, err := config.SessionCollection.InsertOne(context.Background(), session)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create session"})
	}

	session.ID = result.InsertedID.(primitive.ObjectID)
	sessions[session.ID] = &session

	return c.Status(200).JSON(fiber.Map{
		"session_id": session.ID.Hex(),
	})
}

func GetSession(c *fiber.Ctx) error {
	sessionID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session ID"})
	}

	var session models.Session
	err = config.SessionCollection.FindOne(context.Background(), bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Session not found"})
	}

	return c.JSON(fiber.Map{
		"id":         session.ID.Hex(),
		"quiz_id":    session.QuizID.Hex(),
		"creator_id": session.CreatorID.Hex(),
		"participants": func() []string {
			ids := []string{}
			for _, p := range session.Participants {
				ids = append(ids, p.Hex())
			}
			return ids
		}(),
		"is_active": session.IsActive,
		"started":   session.Started,
	})

}

func JoinSession(c *fiber.Ctx) error {
	sessionID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session ID"})
	}

	sess, err := config.Store.Get(c)
	if err != nil {
		return err
	}
	idStr := sess.Get("userID").(string)
	userID, _ := primitive.ObjectIDFromHex(idStr)

	_, err = config.SessionCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": sessionID, "is_active": true, "started": false},
		bson.M{"$addToSet": bson.M{"participants": userID}},
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to join session"})
	}

	if session, ok := sessions[sessionID]; ok {
		session.Participants = append(session.Participants, userID)
	}
	broadcastSessionUpdate(sessionID)
	return c.JSON(fiber.Map{"status": "joined"})
}

func LeaveSession(c *fiber.Ctx) error {
	sessionID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session ID"})
	}

	sess, err := config.Store.Get(c)
	if err != nil {
		return err
	}
	idStr := sess.Get("userID").(string)
	userID, _ := primitive.ObjectIDFromHex(idStr)

	var session models.Session
	err = config.SessionCollection.FindOne(context.Background(), bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Session not found"})
	}

	if session.CreatorID == userID {
		_, err = config.SessionCollection.UpdateOne(
			context.Background(),
			bson.M{"_id": sessionID},
			bson.M{"$set": bson.M{"is_active": false}},
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to close session"})
		}
		filter := bson.M{"_id": sessionID}
		_, _ = config.SessionCollection.DeleteOne(context.Background(), filter)
		delete(sessions, sessionID)
		broadcastSessionClose(sessionID)
	} else {
		_, err = config.SessionCollection.UpdateOne(
			context.Background(),
			bson.M{"_id": sessionID},
			bson.M{"$pull": bson.M{"participants": userID}},
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to leave session"})
		}
		broadcastSessionUpdate(sessionID)
	}

	return c.JSON(fiber.Map{"status": "left"})
}

func StartSession(c *fiber.Ctx) error {
	sessionID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session ID"})
	}

	sess, err := config.Store.Get(c)
	if err != nil {
		return err
	}
	idStr := sess.Get("userID").(string)
	userID, _ := primitive.ObjectIDFromHex(idStr)

	var session models.Session
	err = config.SessionCollection.FindOne(context.Background(), bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Session not found"})
	}

	if len(session.Participants) < 2 {
		return c.Status(400).JSON(fiber.Map{"error": "Not enough participants to start the quiz"})
	}

	if session.CreatorID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Only creator can start the session"})
	}

	_, err = config.SessionCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": sessionID},
		bson.M{"$set": bson.M{"started": true}},
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to start session"})
	}

	if s, ok := sessions[sessionID]; ok {
		s.Started = true
		s.Participants = session.Participants
		broadcastSessionStart(sessionID, s.QuizID.Hex())

		var quiz models.Quiz
		err := config.QuizCollection.FindOne(context.Background(), bson.M{"_id": s.QuizID}).Decode(&quiz)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Quiz not found"})
		}

		runningQuizzes[sessionID] = &RunningQuiz{
			QuizID:    quiz.ID,
			Answers:   make(map[primitive.ObjectID][]bool),
			Answered:  make(map[primitive.ObjectID]bool),
			Finished:  make(map[primitive.ObjectID]bool),
			StartTime: time.Now(),
		}

		connectedUsers[sessionID] = map[primitive.ObjectID]bool{}
		for _, uid := range s.Participants {
			connectedUsers[sessionID][uid] = true
		}

		go startQuestionTimer(sessionID, quiz)
	}

	return c.JSON(fiber.Map{"status": "started"})
}

func SessionWebsocket(c *websocket.Conn) {
	sessionID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		c.Close()
		return
	}

	sessionConnections[sessionID] = append(sessionConnections[sessionID], c)
	defer removeConnection(sessionID, c)

	userIDStr := c.Locals("userID").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err == nil {
		if _, ok := connectedUsers[sessionID]; !ok {
			connectedUsers[sessionID] = map[primitive.ObjectID]bool{}
		}
		connectedUsers[sessionID][userID] = true
	}

	broadcastSessionUpdate(sessionID)

	if rq, ok := runningQuizzes[sessionID]; ok && rq.Current != nil {
		c.WriteJSON(fiber.Map{
			"type": "question",
			"question": fiber.Map{
				"_id":     rq.Current.ID.Hex(),
				"text":    rq.Current.Text,
				"type":    rq.Current.Type,
				"options": rq.Current.Options,
			},
			"timeout": 60,
		})
	}

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

func broadcastSessionUpdate(sessionID primitive.ObjectID) {
	var session models.Session
	err := config.SessionCollection.FindOne(context.Background(), bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		log.Printf("Warning: session not found for broadcast update (sessionID: %v): %v", sessionID.Hex(), err)
		return
	}

	members := []fiber.Map{}
	for _, uid := range session.Participants {
		var user models.User
		err := config.UserCollection.FindOne(context.Background(), bson.M{"_id": uid}).Decode(&user)
		login := "unknown"
		if err == nil {
			login = user.Login
		}

		members = append(members, fiber.Map{
			"userID": uid.Hex(),
			"login":  login,
		})
	}

	msg := fiber.Map{
		"type":    "update",
		"count":   len(session.Participants),
		"members": members,
	}

	if conns, ok := sessionConnections[sessionID]; ok {
		for _, conn := range conns {
			err := conn.WriteJSON(msg)
			if err != nil {
				log.Println("WebSocket write error:", err)
			}
		}
	}
}

func KickFromSession(c *fiber.Ctx) error {
	sessionID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session ID"})
	}

	var body struct {
		UserID string `json:"userId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	removedUserID, err := primitive.ObjectIDFromHex(body.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID to remove"})
	}

	var session models.Session
	err = config.SessionCollection.FindOne(context.Background(), bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Session not found"})
	}

	updateResult, err := config.SessionCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": sessionID},
		bson.M{"$pull": bson.M{"participants": removedUserID}},
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to remove participant"})
	}
	if updateResult.ModifiedCount == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Participant not found in session"})
	}

	if conns, ok := sessionConnections[sessionID]; ok {
		for _, conn := range conns {
			userIDStr := conn.Locals("userID").(string)
			connUserID, _ := primitive.ObjectIDFromHex(userIDStr)
			if connUserID == removedUserID {
				conn.WriteJSON(fiber.Map{
					"type": "kick",
				})
				conn.Close()
				break
			}
		}
	}

	_, err = config.SessionCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": sessionID},
		bson.M{"$pull": bson.M{"participants": removedUserID}},
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to leave session"})
	}
	broadcastSessionUpdate(sessionID)

	return c.JSON(fiber.Map{"status": "participant removed"})
}

func broadcastSessionStart(sessionID primitive.ObjectID, quizID string) {
	if conns, ok := sessionConnections[sessionID]; ok {
		msg := fiber.Map{
			"type":   "start",
			"quizId": quizID,
		}

		for _, conn := range conns {
			conn.WriteJSON(msg)
		}
	}
}

func broadcastSessionClose(sessionID primitive.ObjectID) {
	if conns, ok := sessionConnections[sessionID]; ok {
		msg := fiber.Map{
			"type": "close",
		}

		for _, conn := range conns {
			conn.WriteJSON(msg)
			conn.Close()
		}
		delete(sessionConnections, sessionID)
	}
}

func removeConnection(sessionID primitive.ObjectID, conn *websocket.Conn) {
	if conns, ok := sessionConnections[sessionID]; ok {
		for i, c := range conns {
			if c == conn {
				sessionConnections[sessionID] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
	}
}

func WebsocketUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		sess, err := config.Store.Get(c)
		if err != nil {
			return fiber.ErrUnauthorized
		}
		idStr := sess.Get("userID")
		if idStr == nil {
			return fiber.ErrUnauthorized
		}
		c.Locals("userID", idStr.(string))
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}
