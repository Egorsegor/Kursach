package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Session struct {
	ID           primitive.ObjectID   `json:"_id,omitempty" bson:"_id,omitempty"`
	QuizID       primitive.ObjectID   `json:"quiz_id" bson:"quiz_id"`
	CreatorID    primitive.ObjectID   `json:"creator_id" bson:"creator_id"`
	Participants []primitive.ObjectID `json:"participants" bson:"participants"`
	IsActive     bool                 `json:"is_active" bson:"is_active"`
	Started      bool                 `json:"started" bson:"started"`
}
