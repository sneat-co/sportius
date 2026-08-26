package facade4sportius

import (
	"context"
	"errors"

	sportius "github.com/sneat-co/sneat-ext-contracts/sportius"
)

var (
	// These sentinels preserve errors.Is checks inside the implementation while
	// also exposing the stable cross-repository Sportius error contract.
	ErrInvalid = &sportius.Error{
		Code: sportius.ErrorCodeValidation, MessageKey: "sportius.error.validation",
	}
	ErrNotFound = &sportius.Error{
		Code: sportius.ErrorCodeNotFound, MessageKey: "sportius.error.not_found",
	}
	ErrForbidden = &sportius.Error{
		Code: sportius.ErrorCodeForbidden, MessageKey: "sportius.error.forbidden",
	}
	ErrConflict = &sportius.Error{
		Code: sportius.ErrorCodeConflict, MessageKey: "sportius.error.conflict",
	}
	ErrInvitationExpired = &sportius.Error{
		Code: sportius.ErrorCodeInvitationExpired, MessageKey: "sportius.error.invitation_expired",
	}
)

func invalidField(field, _ string) error {
	return &sportius.Error{
		Code:       sportius.ErrorCodeValidation,
		MessageKey: "sportius.error.validation",
		Field:      field,
		Cause:      ErrInvalid,
	}
}

func notFound(field string, cause error) error {
	return &sportius.Error{
		Code:       sportius.ErrorCodeNotFound,
		MessageKey: "sportius.error.not_found",
		Field:      field,
		Cause:      cause,
	}
}

func conflictf(_ string) error {
	return &sportius.Error{
		Code:       sportius.ErrorCodeConflict,
		MessageKey: "sportius.error.conflict",
		Cause:      ErrConflict,
	}
}

func invitationExpired() error {
	return &sportius.Error{
		Code:       sportius.ErrorCodeInvitationExpired,
		MessageKey: "sportius.error.invitation_expired",
		Cause:      ErrInvitationExpired,
	}
}

func coreFailure(messageKey string, cause error) error {
	return &sportius.Error{
		Code:       sportius.ErrorCodeRetryable,
		MessageKey: messageKey,
		Retryable:  true,
		Cause:      cause,
	}
}

func mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	var contractError *sportius.Error
	if errors.As(err, &contractError) {
		return err
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return &sportius.Error{
			Code:       sportius.ErrorCodeRetryable,
			MessageKey: "sportius.error.retry",
			Retryable:  true,
			Cause:      err,
		}
	}
	return &sportius.Error{
		Code:       sportius.ErrorCodeInternal,
		MessageKey: "sportius.error.internal",
		Cause:      err,
	}
}
