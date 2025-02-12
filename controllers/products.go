package controllers

import (
	"college-bazar-backend/models"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type ProductService struct {
	logger             *zap.Logger
	productsCollection *mongo.Collection
}

func NewProductService(logger *zap.Logger, productsCollection *mongo.Collection) *ProductService {
	return &ProductService{
		logger:             logger,
		productsCollection: productsCollection,
	}
}

func (s *ProductService) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "All products"}`))
}

func (s *ProductService) CreateProduct(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var product models.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	product.ID = primitive.NewObjectID()

	if _, err := s.productsCollection.InsertOne(ctx, product); err != nil {
		s.logger.Error("Failed to insert product", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Product created successfully"}`))
}

func (s *ProductService) GetProductByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := r.URL.Query().Get("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var product models.Product
	err = s.productsCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if errors.Is(err, mongo.ErrNoDocuments) {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	} else if err != nil {
		s.logger.Error("Failed to fetch product", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(product)
}

func (s *ProductService) GetAllProductsOfUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID := r.URL.Query().Get("userId")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	cursor, err := s.productsCollection.Find(ctx, bson.M{"createdBy": objID})
	if err != nil {
		s.logger.Error("Failed to fetch products", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		s.logger.Error("Failed to decode products", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(products)
}

func (s *ProductService) RemoveProduct(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := r.URL.Query().Get("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	res, err := s.productsCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		s.logger.Error("Failed to delete product", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if res.DeletedCount == 0 {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	w.Write([]byte(`{"message": "Product removed successfully"}`))
}
