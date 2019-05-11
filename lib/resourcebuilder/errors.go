package resourcebuilder

type RetryLaterError struct{ Message string }

func (e *RetryLaterError) Error() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return e.Message
}
