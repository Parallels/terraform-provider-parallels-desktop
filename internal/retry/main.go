package retry

import "time"

func For(attempts int, sleep time.Duration, fn func() error) error {
	if err := fn(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			return For(attempts, 2*sleep, fn)
		}

		return err
	}

	return nil
}
