package client

import (
	"fmt"
	"io"
	"time"
)

// Cluster API structure
type Diag struct {
	UploadTime time.Time `json:"upload_time"`
	FileName   string    `json:"filename"`
	ClusterID  string    `json:"cluster_id"`
	HostName   string    `json:"hostname"`
	S3Key      string    `json:"s3_key"`
	Topic      string    `json:"topic"`
	TopicID    string    `json:"topic_id"`
	ID         int       `json:"id"`
	Completed  bool      `json:"completed"`
}

const (
	diagListURLTemplate  = "clusters/%s/support/files"
	diagFileURLTemplate  = "clusters/%s/support/files/%s/content"
	diagFilesURLTemplate = "clusters/%s/support/files/%%s/content" // %%s for passing url template and escaping %s
)

func (client *Client) QueryDiags(clusterID string, options *RequestOptions) (*PagedQuery, error) {
	query, err := client.QueryEntities(
		fmt.Sprintf(diagListURLTemplate, clusterID),
		options)
	if err != nil {
		return nil, err
	}

	return query, nil
}

func (query *PagedQuery) NextDiag() (*Diag, error) {
	diag := &Diag{}
	if len(query.PageResults.Data) == 0 {
		return nil, nil
	}
	ok, err := query.NextEntity(diag)
	if err != nil {
		return nil, fmt.Errorf("failed to get next diag: %w", err)
	}
	if !ok {
		return nil, nil
	}

	return diag, nil
}

func (client *Client) DownloadDiags(clusterID, fileName string) error {
	return client.Download(
		fmt.Sprintf(diagFileURLTemplate,
			clusterID, fileName), fileName)
}

func (client *Client) DownloadManyDiags(clusterID string, fileNames []string) error {
	return client.DownloadMany(
		fmt.Sprintf(diagFilesURLTemplate, clusterID),
		fileNames)
}

func GetDiagsParams(topic, topicID string) *QueryParams {
	params := &QueryParams{}
	if topic != "" {
		params.Set("topic", topic)
	}
	if topicID != "" {
		params.Set("topic_id", topicID)
	}

	return params
}

// ReadDiags reads the content of a diagnostics file and returns a ReadCloser
func (client *Client) ReadDiags(clusterID, file string) (io.ReadCloser, error) {
	return client.Read(
		fmt.Sprintf(diagFileURLTemplate, clusterID, file))
}
