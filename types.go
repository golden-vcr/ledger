package ledger

type TransactionType string

const (
	TransactionTypeManualCredit    TransactionType = "manual-credit"
	TransactionTypeAlertRedemption TransactionType = "alert-redemption"
)
