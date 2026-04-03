package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/rubendubeux/inventory-manager/internal/auth"
	"github.com/rubendubeux/inventory-manager/internal/bot"
	"github.com/rubendubeux/inventory-manager/internal/campaign"
	"github.com/rubendubeux/inventory-manager/internal/category"
	"github.com/rubendubeux/inventory-manager/internal/character"
	"github.com/rubendubeux/inventory-manager/internal/coin"
	"github.com/rubendubeux/inventory-manager/internal/db"
	discordpkg "github.com/rubendubeux/inventory-manager/internal/discord"
	"github.com/rubendubeux/inventory-manager/internal/inventory"
	"github.com/rubendubeux/inventory-manager/internal/shop"
	"github.com/rubendubeux/inventory-manager/internal/transaction"
	"github.com/rubendubeux/inventory-manager/pkg/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading environment variables directly")
	}

	databaseURL := mustEnv("DATABASE_URL")
	port := getEnv("PORT", "8080")
	jwtSecret := mustEnv("JWT_SECRET")
	accessTTL := parseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	refreshTTL := parseDuration(getEnv("JWT_REFRESH_TTL", "168h"))

	pool, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(databaseURL); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	// Auth
	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, auth.Config{
		JWTSecret:       jwtSecret,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	})
	authHandler := auth.NewHandler(authSvc)

	// Campaign
	campaignRepo := campaign.NewRepository(pool)
	campaignSvc := campaign.NewService(campaignRepo)
	campaignHandler := campaign.NewHandler(campaignSvc)

	// Coin
	coinRepo := coin.NewRepository(pool)
	coinSvc := coin.NewService(coinRepo)
	coinHandler := coin.NewHandler(coinSvc)

	// Category
	categoryRepo := category.NewRepository(pool)
	categorySvc := category.NewService(categoryRepo)
	categoryHandler := category.NewHandler(categorySvc)

	// Inventory
	storageRepo := inventory.NewStorageRepository(pool)
	itemRepo := inventory.NewItemRepository(pool)
	coinPurseRepo := inventory.NewCoinRepository(pool)

	// Character (precisa do storageService para EnsureDefaultSpace)
	characterRepo := character.NewRepository(pool)
	characterSvc := character.NewService(characterRepo)

	storageSvc := inventory.NewStorageService(storageRepo, characterRepo)
	itemSvc := inventory.NewItemService(itemRepo, storageRepo, characterRepo, coinRepo, categoryRepo)
	coinPurseSvc := inventory.NewCoinService(coinPurseRepo, characterRepo)
	summarySvc := inventory.NewSummaryService(storageRepo, itemRepo, coinPurseRepo, characterRepo)
	inventoryHandler := inventory.NewHandler(storageSvc, itemSvc, coinPurseSvc, summarySvc)

	characterHandler := character.NewHandler(characterSvc, storageSvc)

	// Shop
	shopRepo := shop.NewRepository(pool)
	shopSvc := shop.NewService(shopRepo, categorySvc, coinSvc)
	shopHandler := shop.NewHandler(shopSvc)

	// Discord
	discordRepo := discordpkg.NewRepository(pool)
	discordSvc := discordpkg.NewService(discordRepo)
	discordHandler := discordpkg.NewHandler(discordSvc)

	// Transaction
	txRepo := transaction.NewRepository(pool)
	txCoinAdapter := &txCoinAdapter{repo: coinPurseRepo}
	txSvc := transaction.NewService(txRepo, characterRepo, shopRepo, categoryRepo, itemRepo, txCoinAdapter, storageRepo, coinSvc)
	txHandler := transaction.NewHandler(txSvc)

	r := chi.NewRouter()
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:3000"}
	if origin := getEnv("ALLOWED_ORIGIN", ""); origin != "" {
		allowedOrigins = append(allowedOrigins, origin)
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/health", healthHandler(pool))

	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/refresh", authHandler.Refresh)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(jwtSecret))
		r.Post("/auth/discord/code", discordHandler.GenerateCode)
		r.Get("/auth/discord/status", discordHandler.Status)
		r.Delete("/auth/discord/link", discordHandler.Unlink)

		r.Post("/campaigns", campaignHandler.Create)
		r.Get("/campaigns", campaignHandler.List)

		r.Post("/campaigns/join", campaignHandler.JoinByCode)

		r.Route("/campaigns/{campaignID}", func(r chi.Router) {
			r.Use(middleware.RequireCampaignRole(pool, "player"))

			r.Get("/", campaignHandler.Get)
			r.Put("/", campaignHandler.Update)
			r.Delete("/", campaignHandler.Delete)

			r.Get("/members", campaignHandler.ListMembers)
			r.Post("/members", campaignHandler.AddMember)
			r.Put("/members/{userID}", campaignHandler.UpdateMember)
			r.Delete("/members/{userID}", campaignHandler.RemoveMember)

			r.Post("/invites", campaignHandler.CreateInvite)

			// Categories
			r.Route("/categories", func(r chi.Router) {
				r.Get("/", categoryHandler.List)
				r.Post("/", categoryHandler.Create)
				r.Route("/{catID}", func(r chi.Router) {
					r.Put("/", categoryHandler.Update)
					r.Delete("/", categoryHandler.Delete)
				})
			})

			// GM Reserve
			r.Get("/reserve", characterHandler.GetReserve)

			// Characters
			r.Route("/characters", func(r chi.Router) {
				r.Get("/", characterHandler.List)
				r.Post("/", characterHandler.Create)

				r.Route("/{charID}", func(r chi.Router) {
					r.Get("/", characterHandler.Get)
					r.Put("/", characterHandler.Update)
					r.Delete("/", characterHandler.Delete)

					// Storage Spaces
					r.Route("/storages", func(r chi.Router) {
						r.Get("/", inventoryHandler.ListStorages)
						r.Post("/", inventoryHandler.CreateStorage)
						r.Route("/{storageID}", func(r chi.Router) {
							r.Put("/", inventoryHandler.UpdateStorage)
							r.Delete("/", inventoryHandler.DeleteStorage)
						})
					})

					// Items
					r.Route("/items", func(r chi.Router) {
						r.Get("/", inventoryHandler.ListItems)
						r.Post("/", inventoryHandler.CreateItem)
						r.Route("/{itemID}", func(r chi.Router) {
							r.Get("/", inventoryHandler.GetItem)
							r.Put("/", inventoryHandler.UpdateItem)
							r.Delete("/", inventoryHandler.DeleteItem)
							r.Post("/transfer", inventoryHandler.TransferItem)
						})
					})

					// Coin Purse
					r.Route("/coins", func(r chi.Router) {
						r.Get("/", inventoryHandler.GetCoinPurse)
						r.Post("/convert", inventoryHandler.ConvertCoins)
						r.Put("/{coinID}", inventoryHandler.SetCoinBalance)
					})

					// Summary
					r.Get("/inventory", inventoryHandler.GetInventorySummary)
					r.Get("/load", inventoryHandler.GetLoad)
				})
			})

			// Coins
			r.Route("/coins", func(r chi.Router) {
				r.Get("/", coinHandler.ListCoinTypes)
				r.Post("/", coinHandler.CreateCoinType)
				r.Get("/default", coinHandler.GetDefaultCoin)

				r.Route("/{coinID}", func(r chi.Router) {
					r.Get("/", coinHandler.GetCoinType)
					r.Put("/", coinHandler.UpdateCoinType)
					r.Delete("/", coinHandler.DeleteCoinType)
					r.Post("/set-default", coinHandler.SetDefaultCoin)
				})
			})

			// Transactions
			r.Route("/transactions", func(r chi.Router) {
				r.Get("/", txHandler.List)
				r.Post("/", txHandler.Create)
				r.Route("/{txID}", func(r chi.Router) {
					r.Get("/", txHandler.Get)
					r.Patch("/", txHandler.Adjust)
					r.Post("/confirm", txHandler.Confirm)
					r.Post("/cancel", txHandler.Cancel)
				})
			})

			// Shops (stores)
			r.Route("/shops", func(r chi.Router) {
				r.Get("/", shopHandler.ListShops)
				r.Post("/", shopHandler.CreateShop)
				r.Route("/{shopID}", func(r chi.Router) {
					r.Put("/", shopHandler.UpdateShop)
					r.Delete("/", shopHandler.DeleteShop)
				})
			})

			// Shop items
			r.Route("/shop", func(r chi.Router) {
				r.Get("/", shopHandler.ListShopItems)
				r.Post("/", shopHandler.CreateShopItem)
				r.Route("/{shopItemID}", func(r chi.Router) {
					r.Get("/", shopHandler.GetShopItem)
					r.Put("/", shopHandler.UpdateShopItem)
					r.Delete("/", shopHandler.DeleteShopItem)
				})
			})

			// Conversions
			r.Route("/conversions", func(r chi.Router) {
				r.Get("/", coinHandler.ListConversions)
				r.Post("/", coinHandler.CreateConversion)
				r.Get("/{conversionID}", coinHandler.GetConversion)
				r.Delete("/{conversionID}", coinHandler.DeleteConversion)
			})
		})
	})

	// Discord bot (opcional — só sobe se DISCORD_TOKEN estiver configurado)
	if discordToken := getEnv("DISCORD_TOKEN", ""); discordToken != "" {
		b, err := bot.New(discordToken, getEnv("DISCORD_GUILD_ID", ""), bot.Deps{
			DiscordSvc:    discordSvc,
			CampaignRepo:  campaignRepo,
			CharacterRepo: characterRepo,
			ItemRepo:      itemRepo,
			StorageRepo:   storageRepo,
			CoinRepo:      coinPurseRepo,
			ShopRepo:      shopRepo,
			CoinTypeRepo:  coinRepo,
			SiteURL:       getEnv("SITE_URL", ""),
		})
		if err != nil {
			log.Printf("bot: erro ao criar cliente Discord: %v", err)
		} else {
			go func() {
				if err := b.Start(); err != nil {
					log.Printf("bot: erro ao iniciar: %v", err)
				}
			}()
			defer b.Stop()
		}
	}

	log.Printf("server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(context.Background()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "db": "unreachable"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "db": "connected"})
	}
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return val
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("invalid duration %q: %v", s, err)
	}
	return d
}

// txCoinAdapter adapts inventory.CoinRepository to transaction.CoinRepo interface.
type txCoinAdapter struct {
	repo *inventory.CoinRepository
}

func (a *txCoinAdapter) GetCoinBalance(ctx context.Context, characterID, coinTypeID uuid.UUID) (float64, error) {
	return a.repo.GetCoinBalance(ctx, characterID, coinTypeID)
}

func (a *txCoinAdapter) ListConversionEdges(ctx context.Context, characterID uuid.UUID) ([]transaction.ConversionEdge, error) {
	edges, err := a.repo.ListAllConversionsForCharacter(ctx, characterID)
	if err != nil {
		return nil, err
	}
	result := make([]transaction.ConversionEdge, len(edges))
	for i, e := range edges {
		result[i] = transaction.ConversionEdge{FromID: e.FromID, ToID: e.ToID, Rate: e.Rate}
	}
	return result, nil
}
