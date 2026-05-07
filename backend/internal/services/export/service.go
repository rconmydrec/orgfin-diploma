package export

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

type Service struct {
	transactionRepo TransactionRepository
	userRepo        UserRepository
	currencySvc     CurrencyConverter
	logger          *slog.Logger
}

func New(
	transactionRepo TransactionRepository,
	userRepo UserRepository,
	currencySvc CurrencyConverter,
	logger *slog.Logger,
) *Service {
	return &Service{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		currencySvc:     currencySvc,
		logger:          logger,
	}
}

// GenerateExcel creates an Excel file with transactions for the given date range.
func (s *Service) GenerateExcel(userID int, startDate, endDate string) ([]byte, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("failed to get user for export", "userID", userID, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	baseCurrencyCode := "USD"
	if user.BaseCurrency != nil {
		baseCurrencyCode = user.BaseCurrency.Code
	}

	transactions, err := s.transactionRepo.GetForExport(userID, startDate, endDate)
	if err != nil {
		s.logger.Error("failed to get transactions for export", "userID", userID, "error", err)
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Transactions"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet: %w", err)
	}
	f.SetActiveSheet(index)
	// Remove the default "Sheet1" if it differs from our sheet name
	if sheetName != "Sheet1" {
		f.DeleteSheet("Sheet1")
	}

	headers := []string{
		"Date", "Description", "Category", "Account",
		"Original Currency", "Amount Original",
		"Base Currency", "Amount Base Currency",
		"Type", "Note",
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#D9E1F2"}, Pattern: 1},
	})

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	totalIncome := decimal.Zero
	totalExpenses := decimal.Zero

	for rowIdx, tx := range transactions {
		row := rowIdx + 2

		// Date
		dateStr := ""
		if tx.DateTime != nil {
			dateStr = tx.DateTime.Format("2006-01-02")
		}
		cell, _ := excelize.CoordinatesToCellName(1, row)
		f.SetCellValue(sheetName, cell, dateStr)

		// Description (Label)
		description := ""
		if tx.Label != nil {
			description = *tx.Label
		}
		cell, _ = excelize.CoordinatesToCellName(2, row)
		f.SetCellValue(sheetName, cell, description)

		// Category
		categoryName := ""
		if tx.Category != nil {
			categoryName = tx.Category.Name
		}
		cell, _ = excelize.CoordinatesToCellName(3, row)
		f.SetCellValue(sheetName, cell, categoryName)

		// Account
		accountName := ""
		if tx.Account != nil {
			accountName = tx.Account.Name
		}
		cell, _ = excelize.CoordinatesToCellName(4, row)
		f.SetCellValue(sheetName, cell, accountName)

		// Original Currency
		originalCurrency := ""
		if tx.Account != nil && tx.Account.Currency != nil {
			originalCurrency = tx.Account.Currency.Code
		}
		cell, _ = excelize.CoordinatesToCellName(5, row)
		f.SetCellValue(sheetName, cell, originalCurrency)

		// Amount Original
		amountOriginal, _ := tx.Amount.Float64()
		cell, _ = excelize.CoordinatesToCellName(6, row)
		f.SetCellValue(sheetName, cell, amountOriginal)

		// Base Currency
		cell, _ = excelize.CoordinatesToCellName(7, row)
		f.SetCellValue(sheetName, cell, baseCurrencyCode)

		// Amount Base Currency - convert using currency service
		txDate := time.Now()
		if tx.DateTime != nil {
			txDate = *tx.DateTime
		}
		amountBase := s.currencySvc.ConvertAmount(tx.Amount, originalCurrency, baseCurrencyCode, txDate)
		amountBaseFloat, _ := amountBase.Float64()
		cell, _ = excelize.CoordinatesToCellName(8, row)
		f.SetCellValue(sheetName, cell, amountBaseFloat)

		// Type
		txType := "Expense"
		if tx.IsIncome {
			txType = "Income"
		}
		cell, _ = excelize.CoordinatesToCellName(9, row)
		f.SetCellValue(sheetName, cell, txType)

		// Note
		note := ""
		if tx.Notes != nil {
			note = *tx.Notes
		}
		cell, _ = excelize.CoordinatesToCellName(10, row)
		f.SetCellValue(sheetName, cell, note)

		// Accumulate totals
		if tx.IsIncome {
			totalIncome = totalIncome.Add(amountBase)
		} else {
			totalExpenses = totalExpenses.Add(amountBase)
		}
	}

	// Summary rows
	summaryRow := len(transactions) + 3
	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})

	// Total Income
	cell, _ := excelize.CoordinatesToCellName(7, summaryRow)
	f.SetCellValue(sheetName, cell, "Total Income:")
	f.SetCellStyle(sheetName, cell, cell, summaryStyle)
	cell, _ = excelize.CoordinatesToCellName(8, summaryRow)
	incomeFloat, _ := totalIncome.Float64()
	f.SetCellValue(sheetName, cell, incomeFloat)
	f.SetCellStyle(sheetName, cell, cell, summaryStyle)

	// Total Expenses
	summaryRow++
	cell, _ = excelize.CoordinatesToCellName(7, summaryRow)
	f.SetCellValue(sheetName, cell, "Total Expenses:")
	f.SetCellStyle(sheetName, cell, cell, summaryStyle)
	cell, _ = excelize.CoordinatesToCellName(8, summaryRow)
	expensesFloat, _ := totalExpenses.Float64()
	f.SetCellValue(sheetName, cell, expensesFloat)
	f.SetCellStyle(sheetName, cell, cell, summaryStyle)

	// Net Balance
	summaryRow++
	cell, _ = excelize.CoordinatesToCellName(7, summaryRow)
	f.SetCellValue(sheetName, cell, "Net Balance:")
	f.SetCellStyle(sheetName, cell, cell, summaryStyle)
	cell, _ = excelize.CoordinatesToCellName(8, summaryRow)
	netBalance := totalIncome.Sub(totalExpenses)
	netBalanceFloat, _ := netBalance.Float64()
	f.SetCellValue(sheetName, cell, netBalanceFloat)
	f.SetCellStyle(sheetName, cell, cell, summaryStyle)

	// Set column widths for readability
	columnWidths := map[string]float64{
		"A": 12, "B": 25, "C": 20, "D": 20,
		"E": 18, "F": 18, "G": 16, "H": 22,
		"I": 10, "J": 30,
	}
	for col, width := range columnWidths {
		f.SetColWidth(sheetName, col, col, width)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write excel to buffer: %w", err)
	}

	return buf.Bytes(), nil
}
