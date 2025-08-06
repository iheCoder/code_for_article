package model

import "fmt"

// Cart Fact

type Cart struct {
	ID         int
	UserID     int
	TotalValue float64
}

func (c Cart) Key() string { return fmt.Sprintf("Cart:%d", c.ID) }
