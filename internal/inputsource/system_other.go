//go:build !darwin

package inputsource

// Other platforms retain the user's selected input source.
func UseSystemKeyboard() (func() error, error) {
	return func() error { return nil }, nil
}
