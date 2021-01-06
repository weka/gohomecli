package client

import "fmt"

func (client *Client) GetAnalytics(clusterID string) ([]byte, error) {
	result := &genericRawResponse{}
	err := client.GetRaw(fmt.Sprintf("clusters/%s/analytics", clusterID), result)
	if err != nil {
		return nil, err
	}
	return result.Data, err
}
