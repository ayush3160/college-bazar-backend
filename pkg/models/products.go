package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Product struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`
	Name          string             `json:"name" bson:"name"`
	Description   string             `json:"description" bson:"description"`
	Image         string             `json:"image" bson:"image"`
	Category      string             `json:"category" bson:"category"`
	Price         float64            `json:"price" bson:"price"`
	WishListCount int                `json:"wishListCount" bson:"wishListCount"`
	CreatedBy     primitive.ObjectID `json:"createdBy" bson:"createdBy"`
}
