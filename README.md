## Exchange Generic OIDC Credentials for GCP Credentials using GCP STS Service


Procedure and referenced library that will exchange an arbitrary OIDC `id_token` for a GCP credential.

You can use the GCP credential then to access any service the mapped principal has GCP IAM permissions on.

The referenced library [github.com/salrashid123/oauth2/google](https://github.com/salrashid123/oauth2#usage-oidc) surfaces an the mapped credential as an [oauth2.TokenSource](https://godoc.org/golang.org/x/oauth2#TokenSource) for use in any GCP cloud library. 

If the underlying credentials expire, this TokenSource will **NOT** automatically renew itself (thats out of scope since its an arbitrary source)

This repo is the second part that explores how to use the [workload identity federation](https://cloud.google.com/iam/docs/access-resources-aws) capability of GCP which allows for external principals (AWS,Azure or arbitrary OIDC provider) to map to a GCP credential.

>> This is not an officially supported Google product

>> `salrashid123/oauth2/google` is also not supported by Google

If you are interested in exchaning AWS credentials for GCP, see

- [Exchange AWS Credentials for GCP Credentials using GCP STS Service](https://github.com/salrashid123/gcpcompat-aws)

---

### Configure OIDC Provider

First we need an OIDC token provider that will give us an `id_token`.  Just for demonstration, we will use [Google Cloud Identity Platform](https://cloud.google.com/identity-platform) as the provider (you can ofcourse use okta, auth0, even google itself).

Setup the identity platform project and acquire an id_token as described in the following tutorial

- [Kubernetes RBAC with Google Cloud Identity Platform/Firebase Tokens](https://github.com/salrashid123/kubernetes_oidc_gcp_identity_platform)

Only up until the step where you generate the token which is all we need to do here.

The GCP project i am using in the example here is called `cicp-oidc`.  Identity platform will automatically create a 'bare bones' oidc `.well-known` endpoint at a url that includes the projectID:

* [https://securetoken.google.com/cicp-oidc/.well-known/openid-configuration](https://securetoken.google.com/cicp-oidc/.well-known/openid-configuration)


Generate the token and notice that the token is for a user called "alice" and her token has the following claims

```bash
export API_KEY=AIzaSyBEHKUoYqPQkQus-redacted

$ python fb_token.py print $API_KEY alice
Getting custom id_token
FB Token for alice
-----------------------------------------------------
Getting STS id_token
STS Token for alice
ID TOKEN: eyJhbGciOiJSUzI1NiIsImtpZCI6ImQxMGM4Zj...
-------
refreshToken TOKEN: AG8BCneX1SdmYipwN-NhJG6dwxbncLT7cuH-redacted
Verified User alice
```

- `id_token`:

```json
{
  "isadmin": "true",
  "groups": [
    "group1",
    "group2"
  ],
  "iss": "https://securetoken.google.com/cicp-oidc",
  "aud": "cicp-oidc",
  "auth_time": 1603538019,
  "user_id": "alice",
  "sub": "alice",
  "iat": 1603538019,
  "exp": 1603541619,
  "firebase": {
    "identities": {},
    "sign_in_provider": "custom"
  }
}
```

Some things to note

* `issuer` is `https://securetoken.google.com/cicp-oidc`,
* `sub` field describes the username
* `isadmin` is a custom claim


### Configure OIDC Federation

We are not ready to configure a GCP Project (which can ofcourse be a different project that the one used for identity platform!)



```bash
export PROJECT_ID=`gcloud config get-value core/project`
export PROJECT_NUMBER=`gcloud projects describe $PROJECT_ID --format='value(projectNumber)'`
```

* Create identity pool

```bash
gcloud beta iam workload-identity-pools create oidc-pool-1 \
    --location="global" \
    --description="OIDC Pool " \
    --display-name="OIDC Pool"
```

* Configure provider
  
  The following command will configure the provider itself.  Notice that we specify the issuer URL without the `.well-known` URL path (since its, well, well-known)

```bash
gcloud beta iam workload-identity-pools providers create-oidc oidc-provider-1 \
    --workload-identity-pool="oidc-pool-1" \
    --issuer-uri="https://securetoken.google.com/cicp-oidc/" \
    --location="global" \
    --attribute-mapping="google.subject=assertion.sub,attribute.isadmin=assertion.isadmin,attribute.aud=assertion.aud" \
    --attribute-condition="attribute.isadmin=='true' && attribute.aud=='cicp-oidc'"
```

  Notice the attribute mapping:
  * `google.subject=assertion.sub`:  This will extract and populate the google subject value from the provided id_token's `sub`  field.
  * `attribute.isadmin=assertion.isadmin`:  This will extract the value of the custom claim `isadmin` and then make it available for IAM rule later as an assertion

  Noticethe attribute conditions:
  * `attribute.isadmin=='true'`: This describes the condition that this provider must meet.  The provided idToken's `isadmin` field MUST be set to true
  * `attribute.aud=='cicp-oidc'`:  This describes the audience value in the token must be set to `cicp-oidc`

If you set the attribute conditions to something else, you should see an error:

```
Unable to exchange token {"error":"unauthorized_client","error_description":"The given credential is rejected by the attribute condition."},
```

* Create GCS Resource

  Create a test GCP resource like GCS and upload a file

```bash
gsutil mb gs://$PROJECT_ID-test
echo fooooo > foo.txt
gsutil cp foo.txt gs://$PROJECT_ID-test
```

* Allow Federated Identity IAM access

  Configure the federated identity access to GCS bucket.

```bash
gcloud projects add-iam-policy-binding $PROJECT_ID  \
 --member "principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/subject/alice" \
 --role roles/storage.objectAdmin
```
 

* Allow Impersonated 

  The type of IAM definition described in the previous step will only work for very few GCP resources (as of 10/24/20).

  At the moment, it will only work for GCS and iamcredentials API. 
  For more information, see [Using Federated or IAM Tokens](https://github.com/salrashid123/gcpcompat-aws#using-federated-or-iam-tokens)
  
  Since it will work for iamcredentials, we can use a federated token to get yet *another* `access_token` that can be used for arbitrary GCP resources.
  That is, use  [generateAccessToken](https://cloud.google.com/iam/docs/reference/credentials/rest/v1/projects.serviceAccounts/generateAccessToken)

  To do this, run a similar IAM rule but this time allow it to impersonate a service account 


```bash
gcloud iam service-accounts create oidc-federated

gcloud iam service-accounts add-iam-policy-binding oidc-federated@$PROJECT_ID.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/subject/alice"
```

  Then allow this service account access to GCS
```bash
gsutil iam ch serviceAccount:oidc-federated@$PROJECT_ID.iam.gserviceaccount.com:objectViewer gs://mineral-minutia-820-cab1
```


### Use OIDC Token

At this point, we are ready to use the OIDC token and exchange it.

Edit `main.go` and specify the variables hardcoded including the id_token from the provider earlier (yes, i'm lazy!)
```golang
	sourceToken := "eyJhbGciOiJSUzI1NiIsImtp..."
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
```

Now run the sample:

```bash
$ go run main.go 
2020/10/24 07:16:14 OIDC Derived GCP access_token: ya29.c.KuQC4gf-xkKbOCIzRGAmAPdL2unF4vLCjZG7TZv7l7bjCK67n2qduIFDs63HR...

fooooo
```

What you should see is the output of the GCS file


Change the value of `UseIAMToken` to true and try running it again.  That flag will either use the federated token directly to access a resource or attempt to exchange it for an IAMCredentials token. 


### Logging

If you used the STS token directly, the principal will appear in the GCS logs
![images/gcs_sts_access.png](images/gcs_sts_access.png)

If you used IAM impersonation, you will see the principalperforming the impersonation

![images/iam_impersonation.png](images/iam_impersonation.png)

and then the impersonated account accessing GCS
![images/gcs_iam_access.png](images/gcs_iam_access.png)


### Organization Policy Restrict

You can also define a GCP [Organization Policy](https://cloud.google.com/resource-manager/docs/organization-policy/creating-managing-policies) that restricts which providers can be enabled for federation

* `constraints/iam.workloadIdentityPoolProviders`

For example, for the following test organization, we will define a policy that only allows you to create a workload identity using
a the specified OIDC providers URL

```bash
$ gcloud organizations list
    DISPLAY_NAME               ID  DIRECTORY_CUSTOMER_ID
    esodemoapp2.com  673208786092              redacted



$ gcloud resource-manager org-policies allow constraints/iam.workloadIdentityPoolProviders \
   --organization=673208786092 https://securetoken.google.com/cicp-oidc/

      constraint: constraints/iam.workloadIdentityPoolProviders
      etag: BwWybJWeyeU=
      listPolicy:
        allowedValues:
        - https://securetoken.google.com/cicp-oidc/
      updateTime: '2020-10-24T15:45:19.794Z'


$ gcloud beta iam workload-identity-pools providers create-oidc oidc-provider-3 \
    --workload-identity-pool="oidc-pool-1" \
    --issuer-uri="https://securetoken.google.com/foo/" \
    --location="global" \
    --attribute-mapping="google.subject=assertion.sub,attribute.isadmin=assertion.isadmin,attribute.aud=assertion.aud" \
    --attribute-condition="attribute.isadmin=='true' && attribute.aud=='cicp-oidc'"
    
    ERROR: (gcloud.beta.iam.workload-identity-pools.providers.create-oidc) FAILED_PRECONDITION: Precondition check failed.
    - '@type': type.googleapis.com/google.rpc.PreconditionFailure
      violations:
      - description: "Org Policy violated for value: 'https://securetoken.google.com/foo/'."
        subject: orgpolicy:projects/user2project2/locations/global/workloadIdentityPools/oidc-pool-1
        type: constraints/iam.workloadIdentityPoolProviders

```