package domain

import "errors"

// Ошибки доставки
//
// Два типа ошибок с разной логикой обработки:
//   TempError — временная: делать ретраи с экспоненциальной задержкой
//   PermError — перманентная: делать ретраи до 3 попыток, после 3-й прекратить

// TempError — временная ошибка доставки.
// Повторить попытку через экспоненциальную задержку.
type TempError struct {
	Msg string
}

func (e *TempError) Error() string {
	return "temporary error: " + e.Msg
}

// IsTempError проверяет является ли ошибка временной.
func IsTempError(err error) bool {
	var e *TempError
	return errors.As(err, &e)
}

// PermError — перманентная ошибка доставки.
type PermError struct {
	Msg string
}

func (e *PermError) Error() string {
	return "permanent error: " + e.Msg
}

// IsPermError проверяет является ли ошибка перманентной.
func IsPermError(err error) bool {
	var e *PermError
	return errors.As(err, &e)
}
