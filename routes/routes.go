package routes

import (
	controllers "college-bazar-backend/controllers"
	middleware "college-bazar-backend/middlewares"
	"net/http"

	"github.com/go-chi/chi"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func New(r chi.Router, logger *zap.Logger, usersCollection, productsCollection, chatsCollection *mongo.Collection) {

	userService := controllers.NewUserService(logger, usersCollection)
	productService := controllers.NewProductService(logger, productsCollection)
	chatService := controllers.NewChatServer(logger, chatsCollection)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", userService.Login)
		r.Post("/register", userService.Register)
	})

	r.Route("/products", func(r chi.Router) {
		r.Use(middleware.JwtMiddleware)
		r.Get("/", productService.GetAllProducts)
		r.Post("/", productService.CreateProduct)
		r.Get("/get", productService.GetProductByID)
	})

	r.Route("/cart", func(r chi.Router) {
		r.Use(middleware.JwtMiddleware)
		r.Get("/", userService.GetCartItems)
		r.Post("/", userService.AddProductToCart)
		r.Delete("/", userService.RemoveProductFromCart)
	})

	r.Route("/ws", func(r chi.Router) {
		r.Use(middleware.JwtMiddleware)
		r.Post("/", chatService.HandleWS)
		r.Get("/chats", chatService.GetMessages)
	})

}
