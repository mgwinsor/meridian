package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/cash"
	"github.com/mgwinsor/meridian/backend/internal/health"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/position"
	"github.com/mgwinsor/meridian/backend/internal/price"
	"github.com/mgwinsor/meridian/backend/internal/property"
)

func newRouter(pool *pgxpool.Pool, healthHandler *health.Handler) http.Handler {
	router := http.NewServeMux()
	healthHandler.RegisterRoutes(router)

	accountRepository := account.NewPostgresRepository(pool)
	accountService := account.NewService(accountRepository)
	accountHandler := account.NewHandler(accountService)
	accountHandler.RegisterRoutes(router)

	cashRepository := cash.NewPostgresRepository(pool)
	cashService := cash.NewService(accountRepository, cashRepository)
	cashHandler := cash.NewHandler(cashService)
	cashHandler.RegisterRoutes(router)

	propertyRepository := property.NewPostgresRepository(pool)
	propertyService := property.NewService(propertyRepository)
	propertyHandler := property.NewHandler(propertyService)
	propertyHandler.RegisterRoutes(router)

	instrumentRepository := instrument.NewPostgresRepository(pool)
	instrumentService := instrument.NewService(instrumentRepository)
	instrumentHandler := instrument.NewHandler(instrumentService)
	instrumentHandler.RegisterRoutes(router)

	positionRepository := position.NewPostgresRepository(pool)
	positionService := position.NewService(accountRepository, instrumentRepository, positionRepository)
	positionHandler := position.NewHandler(positionService)
	positionHandler.RegisterRoutes(router)

	priceRepository := price.NewPostgresRepository(pool)
	priceService := price.NewService(instrumentRepository, priceRepository)
	priceHandler := price.NewHandler(priceService)
	priceHandler.RegisterRoutes(router)

	return router
}
