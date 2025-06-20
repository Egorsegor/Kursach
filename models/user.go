package models

import (
	"Dr-Brain-site-project/config"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID            primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Login         string             `json:"login"`
	Email         string             `json:"email"`
	EmailVerified bool               `json:"emailverified"`
	Password      string             `json:"password"`
}

type TempUser struct {
	Login           string `json:"login"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"passwordConfirm"`
}

func FindUserByID(id string) (User, error) {
	var user User

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user, err
	}

	filter := bson.M{"_id": objectID}
	err = config.UserCollection.FindOne(context.Background(), filter).Decode(&user)

	return user, err
}
