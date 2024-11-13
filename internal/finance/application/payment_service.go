package application

import "github.com/sebuszqo/FinanceManager/internal/finance/domain"

type PaymentService struct {
	repo domain.PaymentRepository
}

func NewPaymentService(repo domain.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) GetAllPaymentMethods() ([]domain.PaymentMethod, error) {
	methods, err := s.repo.GetAllPaymentMethods()
	if err != nil {
		return nil, err
	}

	if methods == nil {
		return []domain.PaymentMethod{}, nil
	}

	return methods, nil
}

func (s *PaymentService) GetUserPaymentSources(userID string) ([]domain.PaymentSource, error) {
	return s.repo.GetUserPaymentSources(userID)
}

func (s *PaymentService) DoesPaymentMethodExistByID(methodID int) (bool, error) {
	return s.repo.PaymentMethodExists(methodID)
}

func (s *PaymentService) DoesUserPaymentSourceExistByID(sourceID int, userID string) (bool, error) {
	return s.repo.UserPaymentSourceExists(sourceID, userID)
}
