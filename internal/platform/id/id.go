package id

import "uuid"

func New() string {
	return uuid.New().String()
}

func IsValid(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
