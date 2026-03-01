// Тесты ошибок доставки TempError/PermError.
// IsTempError — true для TempError и цепочки с TempError, иначе false.
// IsPermError — true для PermError и цепочки с PermError, иначе false.
// TempError_Error / PermError_Error — текст содержит тип и сообщение.
package domain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"notification-processor/internal/domain"
)

func TestIsTempError(t *testing.T) {
	assert.True(t, domain.IsTempError(&domain.TempError{Msg: "timeout"}))
	assert.True(t, domain.IsTempError(errors.Join(errors.New("other"), &domain.TempError{Msg: "x"})))

	assert.False(t, domain.IsTempError(nil))
	assert.False(t, domain.IsTempError(errors.New("plain")))
	assert.False(t, domain.IsTempError(&domain.PermError{Msg: "perm"}))
}

func TestIsPermError(t *testing.T) {
	assert.True(t, domain.IsPermError(&domain.PermError{Msg: "user not found"}))
	assert.True(t, domain.IsPermError(errors.Join(&domain.PermError{Msg: "x"}, errors.New("other"))))

	assert.False(t, domain.IsPermError(nil))
	assert.False(t, domain.IsPermError(errors.New("plain")))
	assert.False(t, domain.IsPermError(&domain.TempError{Msg: "temp"}))
}

func TestTempError_Error(t *testing.T) {
	e := &domain.TempError{Msg: "timeout"}
	assert.Contains(t, e.Error(), "temporary")
	assert.Contains(t, e.Error(), "timeout")
}

func TestPermError_Error(t *testing.T) {
	e := &domain.PermError{Msg: "invalid token"}
	assert.Contains(t, e.Error(), "permanent")
	assert.Contains(t, e.Error(), "invalid token")
}
