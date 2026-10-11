package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spf13/afero"
)

func compareSlice(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for idx, elem := range a {
		if elem != b[idx] {
			return false
		}
	}
	return true
}

// Case empty file.
// Case expected format.
func TestGetAllAWSProfiles(t *testing.T) {
	var tests = []struct {
		fileData       []byte
		expectedOutput []string
		AWSConfirm     bool
		caseName       string
	}{
		{[]byte(""), []string{}, true, "empty file"},
		{[]byte("[default]\naws_access_key=abc\naws_secret_access_key=123\n[123]\naws_access_key=abc\naws_secret_access_key=123"), []string{"default", "123"}, true, "expected format"},
	}
	credentialsFile := config.DefaultSharedCredentialsFilename()

	for _, test := range tests {
		UtilsFs = afero.NewMemMapFs()
		fmt.Printf("[*] Testing %s\n", test.caseName)
		UtilsFs.Create(credentialsFile)
		afero.WriteFile(UtilsFs, credentialsFile, test.fileData, 0755)
		output := GetAllAWSProfiles(test.AWSConfirm)
		if !compareSlice(output, test.expectedOutput) {
			t.Errorf("Test Failed: %v inputted, %v expected, received: %v", test.fileData, test.expectedOutput, output)
		}
	}

}

// Case empty file.
// Case special characters \r \n \t.
// Case with extra new lines at the end.
// Case expected format.
func TestGetSelectedAWSProfiles(t *testing.T) {
	var tests = []struct {
		fileData       []byte
		expectedOutput []string
		AWSConfirm     bool
		caseName       string
	}{
		{[]byte(""), []string{}, true, "empty file"},
		{[]byte("abcd\r\nxyz\t\n123\r\t"), []string{"abcd", "xyz", "123"}, true, "special characters \\r \\n \\t"},
		{[]byte("qwerty\nxyz\n456\n\n\n"), []string{"qwerty", "xyz", "456"}, true, "extra new lines at the end"},
		{[]byte("nmhj\nyuioy\n098"), []string{"nmhj", "yuioy", "098"}, true, "expected format"},
	}

	for _, test := range tests {
		UtilsFs = afero.NewMemMapFs()
		fmt.Printf("[*] Testing %s\n", test.caseName)
		UtilsFs.Create("/tmp/myfile.txt")
		afero.WriteFile(UtilsFs, "/tmp/myfile.txt", test.fileData, 0755)
		output := GetSelectedAWSProfiles("/tmp/myfile.txt")
		if !compareSlice(output, test.expectedOutput) {
			t.Errorf("Test Failed: %v inputted, %v expected, received: %v", test.fileData, test.expectedOutput, output)
		}
	}
}

func TestAWSConfigFileLoaderCustomEndpoint(t *testing.T) {
	const accountID = "123456789012"

	requestURL := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL <- r.URL.String()
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, `<GetCallerIdentityResponse>
	<GetCallerIdentityResult>
		<Account>%s</Account>
		<Arn>arn:aws:iam::%s:user/cloudfox</Arn>
		<UserId>AIDACLOUDFOX</UserId>
	</GetCallerIdentityResult>
</GetCallerIdentityResponse>`, accountID, accountID)
	}))
	t.Cleanup(server.Close)

	originalEndpoint := AWSBaseEndpoint
	originalConfigMap := ConfigMap
	t.Cleanup(func() {
		AWSBaseEndpoint = originalEndpoint
		ConfigMap = originalConfigMap
	})

	AWSBaseEndpoint = server.URL
	ConfigMap = map[string]aws.Config{}
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_ENDPOINT_URL", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	cfg := AWSConfigFileLoader("", "test", "")
	if cfg.BaseEndpoint == nil || aws.ToString(cfg.BaseEndpoint) != server.URL {
		t.Fatalf("expected base endpoint %q, got %v", server.URL, cfg.BaseEndpoint)
	}

	caller, err := sts.NewFromConfig(cfg).GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	if err != nil {
		t.Fatalf("GetCallerIdentity returned an error: %v", err)
	}
	if got := aws.ToString(caller.Account); got != accountID {
		t.Errorf("expected account ID %q, got %q", accountID, got)
	}
	if got := <-requestURL; got != "/" {
		t.Errorf("expected request to custom endpoint root, got %q", got)
	}
}

func TestCustomEndpointSkipsRegionalAWSProbe(t *testing.T) {
	originalEndpoint := AWSBaseEndpoint
	t.Cleanup(func() {
		AWSBaseEndpoint = originalEndpoint
	})

	AWSBaseEndpoint = "http://localhost:4566"
	if !isRegionReachable("not-a-real-region") {
		t.Error("expected a custom endpoint to skip the public regional STS probe")
	}
}
