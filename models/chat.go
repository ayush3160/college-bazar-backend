package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type ChatMessage struct {
	ID         primitive.ObjectID `json:"id" bson:"_id"`
	SenderID   primitive.ObjectID `json:"sender_id" bson:"sender_id"`
	ReceiverID primitive.ObjectID `json:"receiver_id" bson:"receiver_id"`
	Message    string             `json:"message" bson:"message"`
	Timestamp  int64              `json:"timestamp" bson:"timestamp"`
}
