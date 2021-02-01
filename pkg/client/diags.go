package client

import (
	"fmt"
	"time"
)

// Cluster API structure
type Diag struct {
	ID         int       `json:"id"`
	FileName   string    `json:"filename"`
	ClusterID  string    `json:"cluster_id"`
	HostName   string    `json:"hostname"`
	S3Key      string    `json:"s3_key"`
	Completed  bool      `json:"completed"`
	Topic      string    `json:"topic"`
	TopicId    string    `json:"topic_id"`
	UploadTime time.Time `json:"upload_time"`
}

func (client *Client) QueryDiags(clusterID string, options *RequestOptions) (*PagedQuery, error) {
	query, err := client.QueryEntities(
		fmt.Sprintf("clusters/%s/support/files", clusterID),
		options)
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (query *PagedQuery) NextDiag() (*Diag, error) {
	diag := &Diag{}
	ok, err := query.NextEntity(diag)
	if err != nil {
		return nil, fmt.Errorf("failed to get next diag: %s", err)
	}
	if !ok {
		return nil, nil
	}
	return diag, nil
}

func (client *Client) DownloadDiags(clusterID string, fileName string) error {
	return client.Download(
		fmt.Sprintf("clusters/%s/support/files/%s/content", clusterID, fileName),
		fileName,
		&RequestOptions{})
}

func (client *Client) DownloadManyDiags(clusterID string, fileNames []string) error {
	return client.DownloadMany(
		fmt.Sprintf("clusters/%s/support/files/%%s/content", clusterID),
		fileNames,
		&RequestOptions{})
}

func GetDiagBatchParams(topic string, topicId string) *QueryParams {
	return (&QueryParams{}).
		Set("topic", topic).
		Set("topic_id", topicId)
}
