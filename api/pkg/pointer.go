package pkg

func ToPtr[T any](v T) *T {
	return &v
}

func FromPtr[T any](v *T) T {
	return *v
}
