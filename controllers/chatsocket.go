package controllers

import (
	"college-bazar-backend/models"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatMessageRequest struct {
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
}

type ChatServer struct {
	mu             sync.Mutex
	clients        map[primitive.ObjectID]*websocket.Conn
	logger         *zap.Logger
	chatCollection *mongo.Collection
}

func NewChatServer(logger *zap.Logger, chatsCollection *mongo.Collection) *ChatServer {
	return &ChatServer{
		clients:        make(map[primitive.ObjectID]*websocket.Conn),
		logger:         logger,
		chatCollection: chatsCollection,
	}
}

func (s *ChatServer) HandleWS(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(models.UserIDKey).(primitive.ObjectID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade error", zap.Error(err))
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.clients[userID] = conn
	s.mu.Unlock()

	s.logger.Debug("User connected", zap.Any("user_id", userID))

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			s.logger.Error("Read error", zap.Error(err))
			break
		}

		var chatMessageRequest ChatMessageRequest
		if err := json.Unmarshal(msg, &chatMessageRequest); err != nil {
			s.logger.Warn("Invalid message format")
			continue
		}

		chatMessage := models.ChatMessage{
			ID:        primitive.NewObjectID(),
			SenderID:  userID,
			Message:   chatMessageRequest.Message,
			Timestamp: time.Now().Unix(),
		}

		chatMessage.ReceiverID, err = primitive.ObjectIDFromHex(chatMessageRequest.ReceiverID)

		if err != nil {
			s.logger.Warn("Invalid receiver ID")
			continue
		}

		s.saveMessage(chatMessage)

		s.sendMessage(chatMessage)
	}

	s.mu.Lock()
	delete(s.clients, userID)
	s.mu.Unlock()
}

func (s *ChatServer) GetMessages(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(models.UserIDKey).(primitive.ObjectID)

	receiverID := r.URL.Query().Get("receiver_id")
	if receiverID == "" {
		http.Error(w, "receiver_id is required", http.StatusBadRequest)
		return
	}

	receiverObjID, err := primitive.ObjectIDFromHex(receiverID)

	if err != nil {
		http.Error(w, "Invalid receiver_id", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	cursor, err := s.chatCollection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"sender_id": userID, "receiver_id": receiverObjID},
			{"sender_id": receiverObjID, "receiver_id": userID},
		},
		"$sort": bson.M{"timestamp": -1},
	})

	if err != nil {
		s.logger.Error("Failed to fetch messages", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	defer cursor.Close(ctx)

	var messages []models.ChatMessage
	if err := cursor.All(ctx, &messages); err != nil {
		s.logger.Error("Failed to decode messages", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}

func (s *ChatServer) saveMessage(msg models.ChatMessage) {
	_, err := s.chatCollection.InsertOne(context.Background(), msg)
	if err != nil {
		s.logger.Error("Failed to store message", zap.Error(err))
	}
}

func (s *ChatServer) sendMessage(msg models.ChatMessage) {
	s.mu.Lock()
	receiverConn, exists := s.clients[msg.ReceiverID]
	s.mu.Unlock()

	if exists {
		messageData, _ := json.Marshal(msg)
		err := receiverConn.WriteMessage(websocket.TextMessage, messageData)
		if err != nil {
			s.logger.Error("Error sending message", zap.Error(err))
		}
	}
}
