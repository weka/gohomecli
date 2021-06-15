package client

import (
	"fmt"
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
func (client *Client) GetCustomer(id string) (*Customer, error) {
	logger.Info().Str("id", id).Msg("Fetching customer")
	customer := &Customer{}
	err := client.GetAPIEntity("customers", id, customer)
	if err != nil {
		return nil, fmt.Errorf("could not fetch customer %s: %s", id, err)
	}
	return customer, nil
}

func (client *Client) QueryCustomers() (*PagedQuery, error) {
	query, err := client.QueryEntities("customers", &RequestOptions{
		NoAutoFetchNextPage: true,
	})
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (query *PagedQuery) NextCustomer() (*Customer, error) {
	customer := &Customer{}
	ok, err := query.NextEntity(customer)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch next customer: %s", err.Error())
	}
	if !ok {
		return nil, nil
	}
	return customer, nil
}
