package utils

import "testing"

func TestHttp(t *testing.T) {
	config := &FetchConfig{
		Timeout:        10,
		ConnectTimeout: 5,
		MaxRetries:     3,
		RetryDelay:     30,
	}
	var remotes = []string{"https://zip.cm.edu.kg/all.txt", "https://countrymerge.pages.dev/all.txt"}
	list, err := FetchSource(remotes, config)
	if err != nil {
		t.Errorf("error %v", err)
	}
	t.Logf("远程连接: %d", len(list))
}
