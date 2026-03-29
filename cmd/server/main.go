package main

import (
	"luxe-backend/internal/feed"
	"luxe-backend/internal/health"
	"luxe-backend/internal/interaction"
	"luxe-backend/internal/item"
	"luxe-backend/internal/itemhandler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	healthSvc := health.NewService()
	healthHandler := health.NewHandler(healthSvc)
	e.GET("/health", healthHandler.Health)

	interactionRepo := interaction.NewInMemoryRepository()
	interactionSvc := interaction.NewService(interactionRepo)
	interactionHandler := interaction.NewHandler(interactionSvc)
	e.POST("/api/v1/interactions", interactionHandler.Create)

	// Dependency graph: itemhandler → feed → item
	itemRepo := item.NewInMemoryRepository()
	feedSvc := feed.NewFeedService(itemRepo)
	itemHandler := itemhandler.NewHandler(feedSvc)
	e.GET("/api/v1/items", itemHandler.GetItems)

	e.Logger.Fatal(e.Start(":8080"))
}
