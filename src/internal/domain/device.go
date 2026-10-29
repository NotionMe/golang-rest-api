package domain

import "time"

type Device struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	IpAddress string    `json:"ip_address"`
	NodeType  string    `json:"type"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}
