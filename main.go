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

	sourceToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6ImQxMGM4ZjhiMGRjN2Y1NWUyYjM1NDFmMjllNWFjMzc0M2Y3N2NjZWUiLCJ0eXAiOiJKV1QifQ.eyJuYW1lIjoiYWxpY2UiLCJpc2FkbWluIjoidHJ1ZSIsImlzcyI6Imh0dHBzOi8vc2VjdXJldG9rZW4uZ29vZ2xlLmNvbS9jaWNwLW9pZGMtdGVzdCIsImF1ZCI6ImNpY3Atb2lkYy10ZXN0IiwiYXV0aF90aW1lIjoxNjAzNjI0MzAxLCJ1c2VyX2lkIjoiYWxpY2VAZG9tYWluLmNvbSIsInN1YiI6ImFsaWNlQGRvbWFpbi5jb20iLCJpYXQiOjE2MDM2MjQzMDEsImV4cCI6MTYwMzYyNzkwMSwiZW1haWwiOiJhbGljZUBkb21haW4uY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImZpcmViYXNlIjp7ImlkZW50aXRpZXMiOnsiZW1haWwiOlsiYWxpY2VAZG9tYWluLmNvbSJdfSwic2lnbl9pbl9wcm92aWRlciI6InBhc3N3b3JkIn19.oSB2vYLo8gX_CakDaO9MGHYeXGwHUySYYPhhFqL7Fx-glSrQx5O_fMSLqF0p48SvHlN47bNDYfhuwR5HRbxnn_w6XxP0cFkGInRiZngwQyFapiEbpnlT7GCU-u2KWfci0mi770giOBn4ZmiavqtmENZPyR2FcwKCRn9tPNpzFPLXG-uUPjd1zj3YblFsHwBtZo8jcmkDMMo_-Y52z5JQiHyG5sfANjldlgabnygUtInAHNvjJXDiRP0p0u4yuOjjq8mjMX9IPN1KXyHoSqaBjQCVmQqbzlx7jIl75dUxAI7x-OZ-4eZ4fWZvItYaLoQpBHQWpxLszqCYztCKz4dzxg"
	scope := "https://www.googleapis.com/auth/cloud-platform"
	targetResource := "//iam.googleapis.com/projects/540341659919/locations/global/workloadIdentityPools/oidc-pool-1/providers/oidc-provider-1"
	targetServiceAccount := "oidc-federated@cicp-oidc-test.iam.gserviceaccount.com"
	gcpBucketName := "cicp-oidc-test-test"
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
