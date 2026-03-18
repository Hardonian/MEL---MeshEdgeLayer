package security

import (
	"fmt"
	"os"
)

func CheckFileMode(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%s permissions are broader than 0600", path)
	}
	return nil
}
