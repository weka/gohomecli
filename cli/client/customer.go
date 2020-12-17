package client

import (
	"context"
	"time"
)

// Customer API structure
type Customer struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	ImageURL           string    `json:"image_url"`
	Monitored          bool      `json:"monitored"`
	GetWekaIoLastScrub time.Time `json:"get_weka_io_last_scrub"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// GetCustomer returns a single customer
func (client *Client) GetCustomer(ctx context.Context, id string) (*Customer, error) {
	logger.Info().Str("id", id).Msg("Fetching customer")
	customer := &Customer{}
	err := client.getAPIEntity(ctx, "customers", id, customer)
	if err != nil {
		return nil, err
	}
	return customer, nil
}
