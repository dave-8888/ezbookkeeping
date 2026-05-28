package _default

// defaultTransactionDataTSVFileConverter defines the structure of financially_free default tsv file converter
type defaultTransactionDataTSVFileConverter struct {
	defaultTransactionDataPlainTextConverter
}

// Initialize an financially_free default transaction data tsv file converter singleton instance
var (
	DefaultTransactionDataTSVFileConverter = &defaultTransactionDataTSVFileConverter{
		defaultTransactionDataPlainTextConverter{
			columnSeparator: "\t",
		},
	}
)
