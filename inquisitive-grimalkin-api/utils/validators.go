package utils

import (
	"errors"
	"inquisitive-grimalkin/models"
)

func ValidateQuestion(q models.Question) error {
	return nil
}

func ValidateRegistration(u models.User) error {
	if u.Username == "" {
		return errors.New("username cannot be empty")
	}
	if u.Password == "" {
		return errors.New("password cannot be empty")
	}
	// https://stackoverflow.com/questions/201323/how-can-i-validate-an-email-address-using-a-regular-expression
	if u.Email == "" {
		return errors.New("email cannot be empty")
	}
	if u.FirstName == "" {
		return errors.New("first name cannot be empty")
	}
	if u.LastName == "" {
		return errors.New("last name cannot be empty")
	}
	if len(u.Username) > 12 {
		return errors.New("username cannot be longer than 12 characters")
	}
	if len(u.Password) > 36 {
		return errors.New("password cannot be longer than 36 characters")
	}
	
	// TODO: Use Prespective API to remove offensive and bad username, emails, first and last name from being able to be entered into the system

	return nil
}