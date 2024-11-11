package interfaces

import "github.com/sebuszqo/FinanceManager/internal/finance/domain"

type MockPaymentService struct {
	Methods []domain.PaymentMethod
	Err     error
}

func (m *MockPaymentService) ListPaymentMethods() ([]domain.PaymentMethod, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Methods, nil
}

func NewMockPaymentService(methods []domain.PaymentMethod, err error) *MockPaymentService {
	return &MockPaymentService{Methods: methods, Err: err}
}
