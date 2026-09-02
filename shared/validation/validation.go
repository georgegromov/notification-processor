package validation

import (
	"context"

	"github.com/go-playground/validator/v10"
)

type Service struct {
	validate *validator.Validate
}

func NewService() *Service {
	opts := []validator.Option{
		validator.WithRequiredStructEnabled(),
	}

	v := validator.New(opts...)
	return &Service{validate: v}
}

func (v *Service) Validate(ctx context.Context, s any) error {
	return v.validate.StructCtx(ctx, s)
}
