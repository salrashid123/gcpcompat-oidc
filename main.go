package main

import (
	"context"
	"io"
	"log"
	"os"

	"cloud.google.com/go/storage"
	sal "github.com/salrashid123/oauth2/google"
	"google.golang.org/api/option"
)

var ()

func main() {

	sourceToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6ImQxMGM4ZjhiMGRjN2Y1NWUyYjM1NDFmMjllNWFjMzc0M2Y3N2NjZWUiLCJ0eXAiOiJKV1QifQ.eyJpc2FkbWluIjoidHJ1ZSIsImdyb3VwcyI6WyJncm91cDEiLCJncm91cDIiXSwiaXNzIjoiaHR0cHM6Ly9zZWN1cmV0b2tlbi5nb29nbGUuY29tL2NpY3Atb2lkYyIsImF1ZCI6ImNpY3Atb2lkYyIsImF1dGhfdGltZSI6MTYwMzUzODAxOSwidXNlcl9pZCI6ImFsaWNlIiwic3ViIjoiYWxpY2UiLCJpYXQiOjE2MDM1MzgwMTksImV4cCI6MTYwMzU0MTYxOSwiZmlyZWJhc2UiOnsiaWRlbnRpdGllcyI6e30sInNpZ25faW5fcHJvdmlkZXIiOiJjdXN0b20ifX0.aOJiwLcWk8ffuNt-IKGoOvO1JuN3ExIu2ksznsRXZGWPuIQ0SP8nFy-M4JRQQXHDfoLgcBx-SKnblyszeJxKqImKrOlvbZNpjfgb2Jhy4kN-OoPRPUfNRG9qA2FSANnL5mxXxjyuc8XL4NI02pkoIKy1HE0I_tOAXcVvJAg3eTFQzlDX3l4CCwh_N1VraiGWowlGv6sWAQzN0MbSCe7YBJL5CYffKoj_6GaWWX_nbynygbnYLL9AGUmy9o9UlNQTbIoWckZcEkolThddzomiHxYhMmfMP6qEqF3IEDHFyJjp7sPCzulTX1Vv9l6VfXHAuxRKbFMwrSunOu3jJ1GyGg"
	scope := "https://www.googleapis.com/auth/cloud-platform"
	targetResource := "//iam.googleapis.com/projects/1071284184436/locations/global/workloadIdentityPools/oidc-pool-1/providers/oidc-provider-1"
	targetServiceAccount := "oidc-federated@mineral-minutia-820.iam.gserviceaccount.com"
	gcpBucketName := "mineral-minutia-820-cab1"
	gcpObjectName := "foo.txt"

	oTokenSource, err := sal.OIDCFederatedTokenSource(
		&sal.OIDCFederatedTokenConfig{
			SourceToken:          sourceToken,
			Scope:                scope,
			TargetResource:       targetResource,
			TargetServiceAccount: targetServiceAccount,
			UseIAMToken:          true,
		},
	)

	tok, err := oTokenSource.Token()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("OIDC Derived GCP access_token: %s\n", tok.AccessToken)

	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx, option.WithTokenSource(oTokenSource))
	if err != nil {
		log.Fatalf("Could not create storage Client: %v", err)
	}

	bkt := storageClient.Bucket(gcpBucketName)
	obj := bkt.Object(gcpObjectName)
	r, err := obj.NewReader(ctx)
	if err != nil {
		panic(err)
	}
	defer r.Close()
	if _, err := io.Copy(os.Stdout, r); err != nil {
		panic(err)
	}

}
