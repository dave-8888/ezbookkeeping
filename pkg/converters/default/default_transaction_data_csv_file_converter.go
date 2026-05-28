package _default

// defaultTransactionDataCSVFileConverter defines the structure of financially_free default csv file converter
type defaultTransactionDataCSVFileConverter struct {
	defaultTransactionDataPlainTextConverter
}

// Initialize an financially_free default transaction data csv file converter singleton instance
var (
	DefaultTransactionDataCSVFileConverter = &defaultTransactionDataCSVFileConverter{
		defaultTransactionDataPlainTextConverter{
			columnSeparator: ",",
		},
	}
)
