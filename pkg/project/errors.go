package project

import "errors"

var (
	// ErrProjectUnavailable is returned when the Project Service is unavailable.
	ErrProjectUnavailable = errors.New("project service unavailable")

	// ErrProjectTimeout is returned when the Project Service request times out.
	ErrProjectTimeout = errors.New("project service timeout")

	// ErrProjectInvalidResponse is returned when the Project Service returns an invalid response.
	ErrProjectInvalidResponse = errors.New("project service invalid response")

	// ErrProjectUnauthorized is returned when the webhook authentication fails.
	ErrProjectUnauthorized = errors.New("project service unauthorized")
)
