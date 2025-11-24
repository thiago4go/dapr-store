// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020
// Licensed under the MIT License.
//
// Dapr compatible REST API service for products
// ----------------------------------------------------------------------------

package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/thiago4go/dapr-store/cmd/products/ai"
	"github.com/benc-uk/go-rest-api/pkg/problem"
	"github.com/go-chi/chi/v5"
)

// All routes we need should be registered here
func (api API) addRoutes(router chi.Router) {
	router.Get("/get/{id}", api.getProduct)
	router.Get("/catalog", api.getCatalog)
	router.Get("/offers", api.getOffers)
	router.Get("/search/{query}", api.searchProducts)
}

// Return a single product
func (api API) getProduct(resp http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")

	products, err := api.service.QueryProducts("ID", id)
	if err != nil {
		problem.Wrap(500, req.RequestURI, id, err).Send(resp)
		return
	}

	// Handle no results
	if len(products) < 1 {
		problem.Wrap(404, req.RequestURI, id, errors.New("product not found")).Send(resp)
		return
	}

	product := products[0]
	
	// Enhance with AI description if available
	if api.aiClient != nil && api.aiCache != nil {
		product.Description = api.enhanceDescription(req.Context(), product.ID, product.Name, product.Description)
	}

	api.ReturnJSON(resp, product)
}

// Return the product catalog
func (api API) getCatalog(resp http.ResponseWriter, req *http.Request) {
	products, err := api.service.AllProducts()
	if err != nil {
		problem.Wrap(500, req.RequestURI, "catalog", err).Send(resp)
		return
	}

	// Enhance with AI descriptions if available
	if api.aiClient != nil && api.aiCache != nil {
		for i := range products {
			products[i].Description = api.enhanceDescription(req.Context(), products[i].ID, products[i].Name, products[i].Description)
		}
	}

	api.ReturnJSON(resp, products)
}

// Return the products on offer
func (api API) getOffers(resp http.ResponseWriter, req *http.Request) {
	products, err := api.service.QueryProducts("onoffer", "1")
	if err != nil {
		problem.Wrap(500, req.RequestURI, "offers", err).Send(resp)

		return
	}

	api.ReturnJSON(resp, products)
}

// Search the products table
func (api API) searchProducts(resp http.ResponseWriter, req *http.Request) {
	query := chi.URLParam(req, "query")

	products, err := api.service.SearchProducts(query)
	if err != nil {
		problem.Wrap(500, req.RequestURI, query, err).Send(resp)
		return
	}

	api.ReturnJSON(resp, products)
}

// enhanceDescription generates AI description with caching and metrics
func (api API) enhanceDescription(ctx context.Context, productID, productName, currentDesc string) string {
	start := time.Now()
	
	// Check cache first
	cached, err := api.aiCache.Get(ctx, productID)
	if err == nil && cached != "" {
		ai.RecordCacheHit()
		return cached
	}
	
	// Generate new description
	description, err := api.aiClient.GenerateDescription(ctx, productName, currentDesc)
	if err != nil {
		ai.RecordError("generation_failed")
		ai.RecordRequest("error")
		return currentDesc // Graceful fallback
	}
	
	// Cache the result
	if err := api.aiCache.Set(ctx, productID, description); err != nil {
		ai.RecordError("cache_failed")
	}
	
	ai.RecordRequest("success")
	ai.RecordLatency(time.Since(start).Seconds())
	
	return description
}
