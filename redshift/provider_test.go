package redshift

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccProviders map[string]*schema.Provider
	testAccProvider  *schema.Provider
)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"redshift": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func testAccPreCheck(ctx context.Context, t *testing.T) {
	var host string
	if host = os.Getenv("REDSHIFT_HOST"); host == "" {
		t.Fatal("REDSHIFT_HOST must be set for acceptance tests")
	}
	if v := os.Getenv("REDSHIFT_USER"); v == "" {
		t.Fatal("REDSHIFT_USER must be set for acceptance tests")
	}
}

func initTemporaryCredentialsProvider(ctx context.Context, t *testing.T, provider *schema.Provider) {
	clusterIdentifier := getEnvOrSkip("REDSHIFT_TEMPORARY_CREDENTIALS_CLUSTER_IDENTIFIER", t)

	sdkClient, err := stsClient(ctx, t)
	if err != nil {
		t.Skip(fmt.Sprintf("Unable to load STS client due to: %s", err))
	}

	response, err := sdkClient.GetCallerIdentity(ctx, nil)
	if err != nil {
		t.Skip(fmt.Sprintf("Unable to get current STS identity due to: %s", err))
	}
	if response == nil {
		t.Skip("Unable to get current STS identity. Empty response.")
	}

	config := map[string]interface{}{
		"temporary_credentials": []interface{}{
			map[string]interface{}{
				"cluster_identifier": clusterIdentifier,
			},
		},
	}
	if arn, ok := os.LookupEnv("REDSHIFT_TEMPORARY_CREDENTIALS_ASSUME_ROLE_ARN"); ok {
		config["temporary_credentials"].([]interface{})[0].(map[string]interface{})["assume_role"] = []interface{}{
			map[string]interface{}{
				"arn": arn,
			},
		}
	}
	diagnostics := provider.Configure(ctx, terraform.NewResourceConfigRaw(config))
	if diagnostics != nil {
		if diagnostics.HasError() {
			t.Fatalf("Failed to configure temporary credentials provider: %v", diagnostics)
		}
	}
}

func stsClient(ctx context.Context, t *testing.T) (*sts.Client, error) {
	config, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return sts.NewFromConfig(config), nil
}

func TestAccRedshiftTemporaryCredentials(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	provider := Provider()
	assume_role_arn := os.Getenv("REDSHIFT_TEMPORARY_CREDENTIALS_ASSUME_ROLE_ARN")
	defer os.Setenv("REDSHIFT_TEMPORARY_CREDENTIALS_ASSUME_ROLE_ARN", assume_role_arn)
	os.Unsetenv("REDSHIFT_TEMPORARY_CREDENTIALS_ASSUME_ROLE_ARN")
	prepareRedshiftTemporaryCredentialsTestCases(ctx, t, provider)
	client, ok := provider.Meta().(*Client)
	if !ok {
		t.Fatal("Unable to initialize client")
	}
	db, err := client.Connect()
	if err != nil {
		t.Fatalf("Unable to connect to database: %s", err)
	}
	defer db.Close()
}

func TestAccRedshiftTemporaryCredentialsAssumeRole(t *testing.T) {
	_ = getEnvOrSkip("REDSHIFT_TEMPORARY_CREDENTIALS_ASSUME_ROLE_ARN", t)

	ctx, cancel := testContext(t)
	defer cancel()

	provider := Provider()
	prepareRedshiftTemporaryCredentialsTestCases(ctx, t, provider)
	client, ok := provider.Meta().(*Client)
	if !ok {
		t.Fatal("Unable to initialize client")
	}
	db, err := client.Connect()
	if err != nil {
		t.Fatalf("Unable to connect to database: %s", err)
	}
	defer db.Close()
}

func prepareRedshiftTemporaryCredentialsTestCases(ctx context.Context, t *testing.T, provider *schema.Provider) {
	redshift_password := os.Getenv("REDSHIFT_PASSWORD")
	defer os.Setenv("REDSHIFT_PASSWORD", redshift_password)
	os.Unsetenv("REDSHIFT_PASSWORD")
	rawUsername := os.Getenv("REDSHIFT_USER")
	defer os.Setenv("REDSHIFT_USER", rawUsername)
	username := strings.ToLower(permanentUsername(rawUsername))
	os.Setenv("REDSHIFT_USER", username)
	initTemporaryCredentialsProvider(ctx, t, provider)
}
