package gotrans

import (
	"context"
	"fmt"
)

type TranslationRepository interface {
	GetByEntityAndField(
		ctx context.Context,
		locales []Lang,
		entity string,
		entityIDs []int,
	) ([]Translation, error)
}
