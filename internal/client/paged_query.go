package client

import (
	"encoding/json"
)

type PagedQuery struct {
	Client          *Client
	URL             string
	Options         *RequestOptions
	Page            int
	PageResults     queryResultsEnvelope
	HasMorePages    bool
	index           int
	maxIndex        int
	queryMetaParams map[string]interface{}
}

func (client *Client) QueryEntities(url string, options *RequestOptions) (*PagedQuery, error) {
	if options == nil {
		options = &RequestOptions{}
	}
	if options.Params == nil {
		options.Params = make(map[string]interface{})
	}
	query := PagedQuery{
		Client:  client,
		URL:     url,
		Options: options,
		Page:    0,
	}
	err := query.fetchNextPage()
	if err != nil {
		return nil, err
	}
	return &query, nil
}

func (query *PagedQuery) fetchNextPage() error {
	query.Page++
	query.Options.Params["page"] = query.Page
	query.PageResults = queryResultsEnvelope{}
	err := query.Client.Get(query.URL, &query.PageResults, query.Options)
	if err != nil {
		return err
	}
	query.HasMorePages = len(query.PageResults.Data) == query.PageResults.Meta.PageSize
	query.index = -1
	query.maxIndex = len(query.PageResults.Data) - 1
	return nil
}

func (query *PagedQuery) NextEntity(result interface{}) (ok bool, err error) {
	if query.index == query.maxIndex {
		if !query.HasMorePages {
			return false, nil
		}
		err := query.fetchNextPage()
		if err != nil {
			return false, err
		}
	}
	query.index++
	err = json.Unmarshal(query.PageResults.Data[query.index].Attributes, result)
	if err != nil {
		return false, err
	}
	return true, nil
}
