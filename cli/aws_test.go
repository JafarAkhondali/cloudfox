package cli

import "testing"

func TestAWSEndpointURLFlag(t *testing.T) {
	flag := AWSCommands.PersistentFlags().Lookup("endpoint-url")
	if flag == nil {
		t.Fatal("expected persistent endpoint-url flag to be registered")
	}
	if flag.DefValue != "" {
		t.Errorf("expected endpoint-url to default to an empty string, got %q", flag.DefValue)
	}

	const endpoint = "http://localhost:4566"
	originalEndpoint := AWSEndpointURL
	t.Cleanup(func() {
		AWSEndpointURL = originalEndpoint
	})
	if err := flag.Value.Set(endpoint); err != nil {
		t.Fatalf("failed to set endpoint-url flag: %v", err)
	}
	if AWSEndpointURL != endpoint {
		t.Errorf("expected endpoint-url flag to set %q, got %q", endpoint, AWSEndpointURL)
	}
}
