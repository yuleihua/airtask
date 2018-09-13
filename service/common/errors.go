package common

import (
	"errors"
)

var (
	// passphrase is invalid
	ErrInvalidPassword = errors.New("Invalid password")

	// no content
	ErrNOFileContent = errors.New("File content is  null")

	// no file
	ErrNOFile = errors.New("File is  null")

	// no user
	ErrNOUser = errors.New("no user")

	// parameter is invalid
	ErrInvalidParmeter = errors.New("Invalid parameter")

	// passphrase is invalid
	ErrInvalidPasswordPolicy = errors.New("Invalid password. policy: passphrase length is larger than 6; passphrase contains uppercase letters, lowercase letters, and Numbers")
)
