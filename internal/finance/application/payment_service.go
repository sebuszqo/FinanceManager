package application

import "github.com/sebuszqo/FinanceManager/internal/finance/domain"

type PaymentService struct {
	repo domain.PaymentRepository
}

func NewPaymentService(repo domain.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) ListPaymentMethods() ([]domain.PaymentMethod, error) {
	methods, err := s.repo.FindAllPaymentMethods()
	if err != nil {
		return nil, err
	}

	return methods, nil
}
