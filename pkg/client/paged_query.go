package client

import (
	"encoding/json"
)

const defaultPageSize = 50

type PagedQuery struct {
	Client            *Client
	URL               string
	Options           *RequestOptions
	Page              int
	PageResults       queryResultsEnvelope
	noMetaPageResults []json.RawMessage
	HasMorePages      bool
	index             int
	maxIndex          int
	queryMetaParams   map[string]interface{}
}

func (client *Client) QueryEntities(url string, options *RequestOptions) (*PagedQuery, error) {
	if options == nil {
		options = &RequestOptions{}
	}
	if options.Params == nil {
		options.Params = &QueryParams{}
	}
	if options.PageSize == 0 {
		options.PageSize = defaultPageSize
	}
	if options.PageSize > 1000 {
		options.PageSize = 1000
	}
	options.Params.Set("page_size", options.PageSize)
	query := PagedQuery{
		Client:  client,
		URL:     url,
		Options: options,
		Page:    0,
	}
	err := query.FetchNextPage()
	if err != nil {
		return nil, err
	}
	return &query, nil
}

func (query *PagedQuery) FetchNextPage() error {
	query.Page++
	query.Options.Params.Set("page", query.Page)
	var numResultsInPage int
	if query.Options.NoMetadata {
		err := query.Client.Get(query.URL, &query.noMetaPageResults, query.Options)
		if err != nil {
			return err
		}
		numResultsInPage = len(query.noMetaPageResults)
		query.HasMorePages = numResultsInPage == query.Options.PageSize
	} else {
		err := query.Client.Get(query.URL, &query.PageResults, query.Options)
		if err != nil {
			return err
		}
		numResultsInPage = len(query.PageResults.Data)
		query.HasMorePages = numResultsInPage == query.PageResults.Meta.PageSize
	}
	query.index = -1
	query.maxIndex = numResultsInPage - 1
	return nil
}

func (query *PagedQuery) NextEntity(result interface{}) (ok bool, err error) {
	if query.index == query.maxIndex {
		if !query.HasMorePages || query.Options.NoAutoFetchNextPage {
			return false, nil
		}
		err := query.FetchNextPage()
		if err != nil {
			return false, err
		}
	}
	if query.maxIndex <= 0 {
		return false, nil
	}
	query.index++
	if query.Options.NoMetadata {
		err = json.Unmarshal(query.noMetaPageResults[query.index], result)
	} else {
		err = json.Unmarshal(query.PageResults.Data[query.index].Attributes, result)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
