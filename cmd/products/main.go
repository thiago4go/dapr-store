// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020
// Licensed under the MIT License.
//
// Dapr compatible REST API service for products
// ----------------------------------------------------------------------------

package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thiago4go/dapr-store/cmd/products/ai"
	"github.com/thiago4go/dapr-store/cmd/products/impl"
	"github.com/thiago4go/dapr-store/cmd/products/spec"

	"github.com/benc-uk/go-rest-api/pkg/api"
	"github.com/benc-uk/go-rest-api/pkg/env"
	"github.com/benc-uk/go-rest-api/pkg/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "github.com/joho/godotenv/autoload" // Autoloads .env file if it exists
	_ "github.com/mattn/go-sqlite3"
)

// API type is a wrap of the common base API with local implementation
type API struct {
	*api.Base
	service  spec.ProductService
	aiClient *ai.Client
	aiCache  *ai.Cache
}

var (
	healthy     = true               // Simple health flag
	version     = "0.0.1"            // App version number, set at build time with -ldflags "-X 'main.version=1.2.3'"
	buildInfo   = "No build details" // Build details, set at build time with -ldflags "-X 'main.buildInfo=Foo bar'"
	serviceName = "products"
	defaultPort = 9002
)

// Main entry point, will start HTTP service
func main() {
	log.SetOutput(os.Stdout) // Personal preference on log output

	// Port to listen on, change the default as you see fit
	serverPort := env.GetEnvInt("PORT", defaultPort)

	// Use chi for routing
	router := chi.NewRouter()

	// Check if we should use Dapr state store
	daprStoreName := env.GetEnvString("DAPR_STORE_NAME", "")
	
	var service spec.ProductService
	if daprStoreName != "" {
		log.Printf("### Using Dapr state store: %s\n", daprStoreName)
		service = impl.NewDaprService(serviceName, daprStoreName)
		// Initialize products from CSV if needed
		if err := initializeProducts(service); err != nil {
			log.Fatalf("### Failed to initialize products: %v\n", err)
		}
	} else {
		log.Printf("### Using SQLite database\n")
		dbFilePath := "./sqlite.db"
		if len(os.Args) > 1 {
			dbFilePath = os.Args[1]
		}
		service = impl.NewService(serviceName, dbFilePath)
	}

	// Wrapper API with anonymous inner new Base API
	api := API{
		Base:    api.NewBase(serviceName, version, buildInfo, healthy),
		service: service,
	}

	// Initialize AI client if Azure OpenAI is configured
	ctx := context.Background()
	if os.Getenv("AZURE_OPENAI_ENDPOINT") != "" {
		log.Println("### Initializing Azure OpenAI client...")
		aiClient, err := ai.NewClient(ctx)
		if err != nil {
			log.Printf("### Warning: Failed to initialize AI client: %v\n", err)
		} else {
			api.aiClient = aiClient
			// Initialize cache if using Dapr
			if daprSvc, ok := service.(*impl.DaprProductService); ok {
				api.aiCache = ai.NewCache(daprSvc.GetDaprClient())
				log.Println("### AI client and cache initialized (Dapr)")
			} else {
				// Use in-memory cache for SQLite mode
				api.aiCache = ai.NewMemoryCache()
				log.Println("### AI client and cache initialized (in-memory)")
			}
		}
	}

	// Some basic middleware
	router.Use(middleware.RealIP)
	router.Use(logging.NewFilteredRequestLogger(regexp.MustCompile(`(^/metrics)|(^/health)`)))
	router.Use(middleware.Recoverer)
	// Some custom middleware for CORS
	router.Use(api.SimpleCORSMiddleware)
	// Add Prometheus metrics endpoint, must be before the other routes
	api.AddMetricsEndpoint(router, "metrics")

	// Add root, health & status middleware
	api.AddHealthEndpoint(router, "health")
	api.AddStatusEndpoint(router, "status")
	api.AddOKEndpoint(router, "")

	// Add application routes for this service
	api.addRoutes(router)

	// Finally start the server
	api.StartServer(serverPort, router, 5*time.Second)
}

// initializeProducts loads products from CSV if not already initialized
func initializeProducts(service spec.ProductService) error {
	// Check if already initialized
	if _, err := service.QueryProducts("id", "products-initialized"); err == nil {
		log.Println("### Products already initialized")
		return nil
	}

	log.Println("### Initializing products from CSV...")
	csvPath := env.GetEnvString("PRODUCTS_CSV", "./data/products.csv")
	
	file, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Simple CSV parsing (skip header, parse lines)
	// Format: id,name,cost,description,image,onOffer
	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip header
	
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		
		cost, _ := strconv.ParseFloat(parts[2], 64)
		onOffer := parts[5] == "true" || parts[5] == "1"
		
		product := spec.Product{
			ID:          parts[0],
			Name:        parts[1],
			Cost:        float32(cost),
			Description: parts[3],
			Image:       parts[4],
			OnOffer:     onOffer,
		}
		
		// Save via Dapr (this will work with the interface)
		if daprSvc, ok := service.(*impl.DaprProductService); ok {
			if err := daprSvc.SaveProduct(product); err != nil {
				log.Printf("### Warning: Failed to save product %s: %v\n", product.ID, err)
			}
			count++
		}
	}
	
	// Mark as initialized
	if daprSvc, ok := service.(*impl.DaprProductService); ok {
		daprSvc.SaveProduct(spec.Product{ID: "products-initialized"})
	}
	
	log.Printf("### Loaded %d products from CSV\n", count)
	return scanner.Err()
}
