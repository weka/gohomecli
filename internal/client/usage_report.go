package client

import "fmt"

func (client *Client) GetUsageReport(clusterID string) ([]byte, error) {
	result := &genericRawResponse{}
	err := client.GetRaw(fmt.Sprintf("clusters/%s/latest-usage-report", clusterID), result)
	if err != nil {
		return nil, err
	}
	return result.Data, err
}
