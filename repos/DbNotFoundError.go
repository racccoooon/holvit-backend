package repos

type DbNotFoundError struct{}

func (e DbNotFoundError) Error() string {
	return "id not found"
}
