package client

import "fmt"

func (client *Client) GetUsageReport(clusterID string) ([]byte, error) {
	result := &rawResponse{}
	err := client.Get(fmt.Sprintf("clusters/%s/latest-usage-report", clusterID), result, nil)
	if err != nil {
		return nil, err
	}
	return result.Data, err
}
