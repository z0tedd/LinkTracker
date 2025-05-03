package pkg

type RetryableError struct {
	Err error
}

func (e RetryableError) Error() string {
	return e.Err.Error()
}

type TooManyRequestsError struct{}

func (e TooManyRequestsError) Error() string {
	return "too many requests"
}

type UnexpectedError struct{}

func (e UnexpectedError) Error() string {
	return "unexpected error"
}
