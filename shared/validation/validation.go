package validation

import (
	"context"

	"github.com/go-playground/validator/v10"
)

type service struct {
	validate *validator.Validate
}

func NewService() *service {

	opts := []validator.Option{
		validator.WithRequiredStructEnabled(),
	}

	v := validator.New(opts...)
	return &service{validate: v}
}

func (v *service) Validate(ctx context.Context, s any) error {
	return v.validate.StructCtx(ctx, s)
}
