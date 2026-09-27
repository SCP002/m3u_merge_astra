package file

import "os"

// Copy copies <src> file path to <dst> file path
func Copy(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}
	return nil
}
