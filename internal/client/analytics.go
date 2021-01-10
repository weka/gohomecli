package client

import "fmt"

func (client *Client) GetAnalytics(clusterID string) ([]byte, error) {
	result := &rawResponse{}
	err := client.Get(fmt.Sprintf("clusters/%s/analytics", clusterID), result, nil)
	if err != nil {
		return nil, err
	}
	return result.Data, err
}
