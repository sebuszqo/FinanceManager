package application

import (
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"github.com/sebuszqo/FinanceManager/internal/finance/infrastructure"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
	"time"
)

// Helper function to compare floating-point values
func areEqualRounded(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func TestGetTransactionSummary_MultipleYearsMonthsWeeks(t *testing.T) {
	repo := &infrastructure.MockTransactionRepository{
		Transactions: []domain.PersonalTransaction{
			// 2023
			{Date: time.Date(2023, time.January, 10, 0, 0, 0, 0, time.UTC), Type: "income", Amount: 100.12},
			{Date: time.Date(2023, time.January, 15, 0, 0, 0, 0, time.UTC), Type: "expense", Amount: 50.55},
			{Date: time.Date(2023, time.March, 5, 0, 0, 0, 0, time.UTC), Type: "income", Amount: 300.45},
			{Date: time.Date(2023, time.March, 10, 0, 0, 0, 0, time.UTC), Type: "income", Amount: 100.12},
			{Date: time.Date(2023, time.March, 15, 0, 0, 0, 0, time.UTC), Type: "expense", Amount: 75.55},
			{Date: time.Date(2023, time.April, 5, 0, 0, 0, 0, time.UTC), Type: "income", Amount: 200.45},

			// 2022
			{Date: time.Date(2022, time.November, 20, 0, 0, 0, 0, time.UTC), Type: "income", Amount: 150.12},
			{Date: time.Date(2022, time.December, 10, 0, 0, 0, 0, time.UTC), Type: "expense", Amount: 60.55},
			{Date: time.Date(2022, time.December, 25, 0, 0, 0, 0, time.UTC), Type: "income", Amount: 120.45},
			{Date: time.Date(2022, time.December, 30, 0, 0, 0, 0, time.UTC), Type: "expense", Amount: 45.55},

			//  2021
			{Date: time.Date(2021, time.March, 12, 0, 0, 0, 0, time.UTC), Type: "income", Amount: 80.45},
			{Date: time.Date(2021, time.March, 20, 0, 0, 0, 0, time.UTC), Type: "expense", Amount: 30.55},
			{Date: time.Date(2021, time.June, 5, 0, 0, 0, 0, time.UTC), Type: "income", Amount: 50.12},
			{Date: time.Date(2021, time.June, 15, 0, 0, 0, 0, time.UTC), Type: "expense", Amount: 20.55},
		},
	}
	categoryService := &MockCategoryService{}
	paymentService := &PaymentService{}
	service := NewPersonalTransactionService(repo, categoryService, paymentService)

	startDate, _ := time.Parse("2006-01-02", "2021-01-01")
	endDate, _ := time.Parse("2006-01-02", "2023-12-31")

	summary, err := service.GetTransactionSummary("test-user-id", startDate, endDate)
	assert.NoError(t, err)

	year2023 := summary[2023]
	assert.True(t, areEqualRounded(year2023.IncomeTotal, 701.14), fmt.Sprintf("Expected  income total for 2023 to be 700.70, got: %v", year2023.IncomeTotal))
	assert.True(t, areEqualRounded(year2023.ExpenseTotal, 126.1), fmt.Sprintf("Expected  expense total for 2023 to be 126.11, got: %v ", year2023.ExpenseTotal))

	january := year2023.Months["January"]
	assert.True(t, areEqualRounded(january.IncomeTotal, 100.12), fmt.Sprintf("Expected  January 2023 income to be 100.12, got: %v", january.IncomeTotal))
	assert.True(t, areEqualRounded(january.ExpenseTotal, 50.55), fmt.Sprintf("Expected  January 2023 expense to be 50.56, got: %v", january.ExpenseTotal))

	march := year2023.Months["March"]
	assert.True(t, areEqualRounded(march.IncomeTotal, 400.58), fmt.Sprintf("Expected  March 2023 income to be 400.58, got: %v", march.IncomeTotal))
	assert.True(t, areEqualRounded(march.ExpenseTotal, 75.55), fmt.Sprintf("Expected  March 2023 expense to be 75.56, got: %v", march.ExpenseTotal))

	april := year2023.Months["April"]
	assert.True(t, areEqualRounded(april.IncomeTotal, 200.45), fmt.Sprintf("Expected  April 2023 income to be 200.46, got: %v", april.IncomeTotal))
	assert.True(t, areEqualRounded(april.ExpenseTotal, 0), fmt.Sprintf("Expected  April 2023 expense to be 0, got: %v", april.ExpenseTotal))

	year2022 := summary[2022]
	assert.True(t, areEqualRounded(year2022.IncomeTotal, 270.58), fmt.Sprintf("Expected  income total for 2022 to be 270.58, got: %v", year2022.IncomeTotal))
	assert.True(t, areEqualRounded(year2022.ExpenseTotal, 106.1), fmt.Sprintf("Expected  expense total for 2022 to be 106.11, got: %v", year2022.ExpenseTotal))

	december := year2022.Months["December"]
	assert.True(t, areEqualRounded(december.IncomeTotal, 120.46), fmt.Sprintf("Expected  December 2022 income to be 120.46, got: %v", december.IncomeTotal))
	assert.True(t, areEqualRounded(december.ExpenseTotal, 106.1), fmt.Sprintf("Expected  December 2022 expense to be 106.11, got: %v", december.ExpenseTotal))

	year2021 := summary[2021]
	assert.True(t, areEqualRounded(year2021.IncomeTotal, 130.57), fmt.Sprintf("Expected  income total for 2021 to be 130.58, got: %v", year2021.IncomeTotal))
	assert.True(t, areEqualRounded(year2021.ExpenseTotal, 51.11), fmt.Sprintf("Expected  expense total for 2021 to be 51.11, got: %v", year2021.ExpenseTotal))

	march2021 := year2021.Months["March"]
	assert.True(t, areEqualRounded(march2021.IncomeTotal, 80.46), fmt.Sprintf("Expected  March 2021 income to be 80.46, got: %v", march2021.IncomeTotal))
	assert.True(t, areEqualRounded(march2021.ExpenseTotal, 30.56), fmt.Sprintf("Expected  March 2021 expense to be 30.56, got: %v", march2021.ExpenseTotal))

	june2021 := year2021.Months["June"]
	assert.True(t, areEqualRounded(june2021.IncomeTotal, 50.12), fmt.Sprintf("Expected  June 2021 income to be 50.12, got: %v", june2021.IncomeTotal))
	assert.True(t, areEqualRounded(june2021.ExpenseTotal, 20.56), fmt.Sprintf(fmt.Sprintf("Expected  June 2021 expense to be 20.56, got: %v", june2021.ExpenseTotal)))
}
