package client

import (
	"encoding/json"
)

const defaultPageSize = 50

type PagedQuery struct {
	Client            *Client
	Options           *RequestOptions
	queryMetaParams   map[string]any
	URL               string
	nextCursor        string
	noMetaPageResults []json.RawMessage
	PageResults       queryResultsEnvelope
	Page              int
	index             int
	maxIndex          int
	HasMorePages      bool
	useCursor         bool
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
	if !options.NoMetadata {
		options.Params.Set("meta", true)
	}
	query := PagedQuery{
		Client:    client,
		URL:       url,
		Options:   options,
		useCursor: true,
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
	if query.useCursor && len(query.nextCursor) > 0 {
		query.Options.Params.Set("cursor", query.nextCursor)
	}
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
		if query.useCursor {
			query.nextCursor = query.PageResults.Meta.NextCursor
			query.HasMorePages = query.PageResults.Meta.HasNextPage
			numResultsInPage = len(query.PageResults.Entries)
		} else {
			numResultsInPage = len(query.PageResults.Data)
			query.HasMorePages = numResultsInPage == query.PageResults.Meta.PageSize
		}
	}
	query.index = -1
	query.maxIndex = numResultsInPage - 1

	return nil
}

func (query *PagedQuery) NextEntity(result any) (ok bool, err error) {
	if query.index == query.maxIndex {
		if !query.HasMorePages || query.Options.NoAutoFetchNextPage {
			return false, nil
		}
		err := query.FetchNextPage()
		if err != nil {
			return false, err
		}
	}
	if query.maxIndex < 0 {
		return false, nil
	}
	query.index++
	switch {
	case query.Options.NoMetadata:
		err = json.Unmarshal(query.noMetaPageResults[query.index], result)
	case query.useCursor:
		err = json.Unmarshal(query.PageResults.Entries[query.index], result)
	default:
		err = json.Unmarshal(query.PageResults.Data[query.index].Attributes, result)
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
