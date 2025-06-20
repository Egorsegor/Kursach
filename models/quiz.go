package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type QuestionType string

const (
	QuestionTypeTrueFalse      QuestionType = "true_false"
	QuestionTypeSingleChoice   QuestionType = "single_choice"
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
)

type Question struct {
	ID            primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Text          string             `json:"text"`
	Type          QuestionType       `json:"type"`
	Options       []string           `json:"options"`
	CorrectAnswer []int              `json:"correctAnswer"`
}

type Quiz struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"userid"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Questions   []Question         `json:"questions"`
}

func CompareAnswers(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	diff := make(map[int]int, len(a))

	for _, num := range a {
		diff[num]++
	}

	for _, num := range b {
		if diff[num] == 0 {
			return false
		}
		diff[num]--
	}

	return true
}
