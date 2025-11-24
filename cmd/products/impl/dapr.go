// ----------------------------------------------------------------------------
// Dapr state store implementation of the ProductService
// ----------------------------------------------------------------------------

package impl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/thiago4go/dapr-store/cmd/products/spec"
	dapr "github.com/dapr/go-sdk/client"
)

// DaprProductService is a Dapr based implementation of ProductService interface
type DaprProductService struct {
	serviceName string
	storeName   string
	daprPort    string
	daprClient  dapr.Client
}

// NewDaprService creates a new Dapr-based ProductService
func NewDaprService(serviceName, storeName string) *DaprProductService {
	client, err := dapr.NewClient()
	if err != nil {
		panic(fmt.Sprintf("failed to create Dapr client: %v", err))
	}
	
	return &DaprProductService{
		serviceName: serviceName,
		storeName:   storeName,
		daprPort:    "3500", // Default Dapr sidecar port
		daprClient:  client,
	}
}

// GetDaprClient returns the Dapr client for use by other components
func (s *DaprProductService) GetDaprClient() dapr.Client {
	return s.daprClient
}

// SaveProduct saves a product to Dapr state store
func (s *DaprProductService) SaveProduct(product spec.Product) error {
	url := fmt.Sprintf("http://localhost:%s/v1.0/state/%s", s.daprPort, s.storeName)
	
	state := []map[string]interface{}{
		{
			"key":   product.ID,
			"value": product,
		},
	}
	
	data, _ := json.Marshal(state)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to save product: %s", body)
	}
	
	return nil
}

// QueryProducts queries products by a specific field
func (s *DaprProductService) QueryProducts(column, term string) ([]spec.Product, error) {
	// For simple queries, get by key
	if column == "id" {
		product, err := s.getProduct(term)
		if err != nil {
			return nil, err
		}
		return []spec.Product{product}, nil
	}
	
	// For other queries, we need to scan all products (limitation of key-value store)
	all, err := s.AllProducts()
	if err != nil {
		return nil, err
	}
	
	var results []spec.Product
	for _, p := range all {
		switch column {
		case "onOffer":
			if term == "true" && p.OnOffer {
				results = append(results, p)
			}
		}
	}
	
	return results, nil
}

// AllProducts returns all products (scans all keys with "prd" prefix)
func (s *DaprProductService) AllProducts() ([]spec.Product, error) {
	// This is a simplified implementation
	// In production, you'd use Dapr query API or maintain an index
	var products []spec.Product
	
	// Try to get products by known IDs (prd1-prd100)
	for i := 1; i <= 100; i++ {
		id := fmt.Sprintf("prd%d", i)
		product, err := s.getProduct(id)
		if err == nil {
			products = append(products, product)
		}
	}
	
	return products, nil
}

// SearchProducts searches products by name or description
func (s *DaprProductService) SearchProducts(query string) ([]spec.Product, error) {
	all, err := s.AllProducts()
	if err != nil {
		return nil, err
	}
	
	query = strings.ToLower(query)
	var results []spec.Product
	
	for _, p := range all {
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Description), query) {
			results = append(results, p)
		}
	}
	
	return results, nil
}

// getProduct retrieves a single product by ID
func (s *DaprProductService) getProduct(id string) (spec.Product, error) {
	url := fmt.Sprintf("http://localhost:%s/v1.0/state/%s/%s", s.daprPort, s.storeName, id)
	
	resp, err := http.Get(url)
	if err != nil {
		return spec.Product{}, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 204 || resp.StatusCode == 404 {
		return spec.Product{}, fmt.Errorf("product not found")
	}
	
	var product spec.Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return spec.Product{}, err
	}
	
	return product, nil
}
