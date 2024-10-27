package portfolios

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"time"
)

var (
	ErrPortfolioNotFound  = errors.New("portfolio not found")
	ErrUnauthorizedAccess = errors.New("unauthorized: user does not own this portfolio")
	ErrPortfolioNameTaken = errors.New("portfolio with this name already exists")
)

type Service interface {
	CreatePortfolio(ctx context.Context, userID string, name, description string) (*Portfolio, error)
	GetPortfolio(ctx context.Context, portfolioID uuid.UUID, userID string) (*Portfolio, error)
	GetAllPortfolios(ctx context.Context, userID string) ([]PortfolioDTO, error)
	UpdatePortfolio(ctx context.Context, portfolioID uuid.UUID, userID string, name, description *string) error
	DeletePortfolio(ctx context.Context, portfolioID uuid.UUID, userID string) error
	CheckPortfolioOwnership(ctx context.Context, portfolioID uuid.UUID, userID string) (bool, error)
}

type service struct {
	portfolioRepo PortfolioRepository
}

func NewPortfolioService(repo PortfolioRepository) Service {
	return &service{portfolioRepo: repo}
}

func (s *service) CreatePortfolio(ctx context.Context, userID string, name, description string) (*Portfolio, error) {
	exists, err := s.portfolioRepo.ExistsByName(ctx, userID, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrPortfolioNameTaken
	}
	portfolio := &Portfolio{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = s.portfolioRepo.Create(ctx, portfolio)
	return portfolio, err
}

func (s *service) GetPortfolio(ctx context.Context, portfolioID uuid.UUID, userID string) (*Portfolio, error) {
	var portfolio Portfolio
	err := s.portfolioRepo.FindByID(ctx, portfolioID, &portfolio)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPortfolioNotFound
		}
		return nil, err
	}
	if portfolio.UserID != userID {
		return nil, ErrUnauthorizedAccess
	}
	return &portfolio, nil
}

func (s *service) UpdatePortfolio(ctx context.Context, portfolioID uuid.UUID, userID string, name, description *string) error {
	portfolio := &Portfolio{}
	err := s.portfolioRepo.FindByID(ctx, portfolioID, portfolio)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPortfolioNotFound
		}
		return err
	}

	if portfolio.UserID != userID {
		return ErrUnauthorizedAccess
	}

	if name != nil && *name != portfolio.Name {
		exists, err := s.portfolioRepo.ExistsByName(ctx, userID, *name)
		if err != nil {
			return err
		}
		if exists {
			return ErrPortfolioNameTaken
		}
		portfolio.Name = *name
	}

	if description != nil {
		portfolio.Description = *description
	}

	affected, err := s.portfolioRepo.Update(ctx, portfolio)
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrPortfolioNotFound
	}
	return nil
}

func (s *service) DeletePortfolio(ctx context.Context, portfolioID uuid.UUID, userID string) error {
	portfolio := &Portfolio{}
	err := s.portfolioRepo.FindByID(ctx, portfolioID, portfolio)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPortfolioNotFound
		}
		return err
	}

	if portfolio.UserID != userID {
		return ErrUnauthorizedAccess
	}
	err = s.portfolioRepo.DeletePortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}
	return nil
}
func (s *service) GetAllPortfolios(ctx context.Context, userID string) ([]PortfolioDTO, error) {
	var portfolioList []PortfolioDTO
	err := s.portfolioRepo.findAllByUserID(ctx, userID, &portfolioList)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPortfolioNotFound
		}
		return nil, err
	}

	return portfolioList, nil
}

func (s *service) CheckPortfolioOwnership(ctx context.Context, portfolioID uuid.UUID, userID string) (bool, error) {
	portfolio := &Portfolio{}
	err := s.portfolioRepo.FindByID(ctx, portfolioID, portfolio)
	if err != nil {
		return false, err
	}
	return portfolio.UserID == userID, nil
}
