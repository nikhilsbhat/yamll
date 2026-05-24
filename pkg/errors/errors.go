package errors

func (e *YamlError) Error() string {
	return e.Message
}

func (e *YamllError) Error() string {
	return e.Message
}

func (e *YamlEmptyError) Error() string {
	return e.Message
}
