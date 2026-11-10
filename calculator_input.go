package main

import "github.com/egoist/mygo"

// calculatorInput is owned by the native UI thread. Page and window focus
// jointly own the temporary keyboard selection; losing either releases it.
type calculatorInput struct {
	page, focused  bool
	selectKeyboard func() (func() error, error)
	restore        func() error
}

func (c *calculatorInput) sync() error {
	if c.page && c.focused {
		if c.restore == nil {
			restore, err := c.selectKeyboard()
			if err != nil {
				return err
			}
			c.restore = restore
		}
		return nil
	}
	if c.restore != nil {
		if err := c.restore(); err != nil {
			return err
		}
		c.restore = nil
	}
	return nil
}

func (s *System) SetCalculatorInputActive(active bool) (err error) {
	mygo.RunOnMain(func() {
		s.calculatorInput.page = active
		err = s.calculatorInput.sync()
	})
	return err
}
