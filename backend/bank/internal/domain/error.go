package domain

type ErrorMessageType string

type BankAccountError struct {
	Type    ErrorMessageType
	Message string
}

const (
	ErrorTypeNotFound      ErrorMessageType = "NotFound"
	ErrorTypeAlreadyExists ErrorMessageType = "AlreadyExists"
	ErrorTypeInconsistent  ErrorMessageType = "InconsistentState"
)

func NewBankAccountError(t ErrorMessageType, msg string) *BankAccountError {
	return &BankAccountError{Type: t, Message: msg}
}

func (e *BankAccountError) Error() string {
	return e.Message
}
