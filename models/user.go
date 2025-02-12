package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID   `json:"id" bson:"_id"`
	Username string               `json:"username" bson:"username"`
	Password string               `json:"password" bson:"password"`
	Email    string               `json:"email" bson:"email"`
	Image    string               `json:"image" bson:"image"`
	Cart     []primitive.ObjectID `json:"cart" bson:"cart"`
}

type userID string

var (
	UserIDKey userID = "userID"
)
