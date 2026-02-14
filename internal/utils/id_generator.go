package utils

import "github.com/google/uuid"

func GenerateUUID() uuid.UUID {
	val, err := uuid.NewV7()
	if err != nil {
		panic("failed to generate UUID: " + err.Error())
	}
	return val
}
