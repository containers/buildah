package buildah

// GenericPullError is an explanatory text for rejecting a pull of an image.
type GenericPullError struct {
	message string
}

// NewGenericPullError is used to create a new GenericPullError
func NewGenericPullError(message string) GenericPullError {
	return GenericPullError{
		message: message,
	}
}

// Error returns the message from a GenericPullError
func (e GenericPullError) Error() string {
	return e.message
}
