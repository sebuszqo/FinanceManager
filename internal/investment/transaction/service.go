package transactions

import (
	"context"
	"github.com/google/uuid"
	"github.com/sebuszqo/FinanceManager/internal/investment/models"
	"log"
	"sync"
	"time"
)

type TransactionType struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

type Service interface {
	SetAssetService(assetService AssetService)
	CreateTransaction(ctx context.Context, assetID uuid.UUID, userID string, transaction *models.Transaction) error
	GetTransactionTypes() []TransactionType
	GetAllTransactions(ctx context.Context, assetID uuid.UUID) ([]models.Transaction, error)
}

type AssetService interface {
	UpdateAssetAggregates(ctx context.Context, assetID uuid.UUID) error
	// Other methods...
}

type service struct {
	transactionRepo      TransactionRepository
	assetService         AssetService
	transactionTypeCache map[int]string
	mu                   sync.RWMutex
}

func NewTransactionService(repo TransactionRepository) Service {
	service := &service{
		transactionRepo:      repo,
		transactionTypeCache: make(map[int]string),
	}

	if err := service.loadTransactionTypesIntoCache(context.Background()); err != nil {
		log.Fatalf("Failed to load transaction types into cache: %v", err)
	}

	go service.startPeriodicCacheRefresh(10 * time.Minute)
	return service
}

func (s *service) SetAssetService(assetService AssetService) {
	s.assetService = assetService
}

func (s *service) startPeriodicCacheRefresh(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			if err := s.loadTransactionTypesIntoCache(context.Background()); err != nil {
				log.Println("Failed to refresh transaction types cache:", err)
			} else {
				log.Println("Transaction types cache refreshed")
			}
		}
	}
}

func (s *service) loadTransactionTypesIntoCache(ctx context.Context) error {
	transactionTypes, err := s.transactionRepo.getTransactionTypes(ctx) // Fetch transaction types
	if err != nil {
		return err
	}

	// Assuming you are using some kind of cache here
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tt := range transactionTypes {
		s.transactionTypeCache[tt.ID] = tt.Type // Storing them in cache
	}

	return nil
}

func (s *service) GetTransactionTypes() []TransactionType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert cache map back to slice
	transactionTypes := make([]TransactionType, 0, len(s.transactionTypeCache))
	for id, t := range s.transactionTypeCache {
		transactionTypes = append(transactionTypes, TransactionType{ID: id, Type: t})
	}
	return transactionTypes
}

func (s *service) CreateTransaction(ctx context.Context, assetID uuid.UUID, userID string, transaction *models.Transaction) error {
	err := s.transactionRepo.create(ctx, transaction)
	if err != nil {
		return err
	}

	err = s.assetService.UpdateAssetAggregates(ctx, assetID)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) GetAllTransactions(ctx context.Context, assetID uuid.UUID) ([]models.Transaction, error) {
	return s.transactionRepo.getTransactionsByAsset(ctx, assetID)
}
