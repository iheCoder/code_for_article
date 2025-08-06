package model

import "fmt"

// User Fact

type User struct {
	ID    int
	Level string
}

func (u User) Key() string { return fmt.Sprintf("User:%d", u.ID) }
