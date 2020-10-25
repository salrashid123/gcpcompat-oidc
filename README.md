## Exchange Generic OIDC Credentials for GCP Credentials using GCP STS Service


Procedure and referenced library that will exchange an arbitrary OIDC `id_token` for a GCP credential.

You can use the GCP credential then to access any service the mapped principal has GCP IAM permissions on.

The referenced library [github.com/salrashid123/oauth2/google](https://github.com/salrashid123/oauth2#usage-oidc) surfaces an the mapped credential as an [oauth2.TokenSource](https://godoc.org/golang.org/x/oauth2#TokenSource) for use in any GCP cloud library. 

If the underlying credentials expire, this TokenSource will **NOT** automatically renew itself (thats out of scope since its an arbitrary source)

This repo is the second part that explores how to use the [workload identity federation](https://cloud.google.com/iam/docs/access-resources-aws) capability of GCP which allows for external principals (AWS,Azure or arbitrary OIDC provider) to map to a GCP credential.

>> This is not an officially supported Google product

>> `salrashid123/oauth2/google` is also not supported by Google

If you are interested in exchanging AWS credentials for GCP, see

- [Exchange AWS Credentials for GCP Credentials using GCP STS Service](https://github.com/salrashid123/gcpcompat-aws)

---

### Configure OIDC Provider

First we need an OIDC token provider that will give us an `id_token`.  Just for demonstration, we will use [Google Cloud Identity Platform](https://cloud.google.com/identity-platform) as the provider (you can ofcourse use okta, auth0, even google itself).

Setup the identity platform project and acquire an id_token as described in at the *end* of this repo

The GCP project i am using in the example here is called `cicp-oidc-test`.  Identity platform will automatically create a 'bare bones' oidc `.well-known` endpoint at a url that includes the projectID:

* [https://securetoken.google.com/cicp-oidc-test/.well-known/openid-configuration](https://securetoken.google.com/cicp-oidc/.well-known/openid-configuration)


Generate the token and notice that the token is for a user called "alice" and her token has the following claims

- `id_token`:

```json
  {
    "name": "alice",
    "isadmin": "true",
    "iss": "https://securetoken.google.com/cicp-oidc-test",
    "aud": "cicp-oidc-test",
    "auth_time": 1603624301,
    "user_id": "alice@domain.com",
    "sub": "alice@domain.com",
    "iat": 1603624301,
    "exp": 1603627901,
    "email": "alice@domain.com",
    "email_verified": true,
    "firebase": {
      "identities": {
        "email": [
          "alice@domain.com"
        ]
      },
      "sign_in_provider": "password"
    }
  }
```

Some things to note

* `issuer` is `https://securetoken.google.com/cicp-oidc-test`,
* `sub` field describes the username
* `isadmin` is a custom claim


### Configure OIDC Federation

Just for demonstration, we will configure one gcp project that will own the identity provider (GCP Identity Platform) and the GCP federation that allows it access to a resource.  You can ofcourse use different projects or any other identity provider that supports OIDC.


```bash
export PROJECT_ID=`gcloud config get-value core/project`
export PROJECT_NUMBER=`gcloud projects describe $PROJECT_ID --format='value(projectNumber)'`
```

* Create identity pool

```bash
gcloud beta iam workload-identity-pools create oidc-pool-1 \
    --location="global" \
    --description="OIDC Pool " \
    --display-name="OIDC Pool" --project $PROJECT_ID
```

* Configure provider
  
  The following command will configure the provider itself.  Notice that we specify the issuer URL without the `.well-known` URL path (since its, well, well-known)

```bash
gcloud beta iam workload-identity-pools providers create-oidc oidc-provider-1 \
    --workload-identity-pool="oidc-pool-1" \
    --issuer-uri="https://securetoken.google.com/cicp-oidc-test/" \
    --location="global" \
    --attribute-mapping="google.subject=assertion.sub,attribute.isadmin=assertion.isadmin,attribute.aud=assertion.aud" \
    --attribute-condition="attribute.isadmin=='true' && attribute.aud=='cicp-oidc-test'" --project $PROJECT_ID
```

  Notice the attribute mapping:
  * `google.subject=assertion.sub`:  This will extract and populate the google subject value from the provided id_token's `sub`  field.
  * `attribute.isadmin=assertion.isadmin`:  This will extract the value of the custom claim `isadmin` and then make it available for IAM rule later as an assertion

  Noticethe attribute conditions:
  * `attribute.isadmin=='true'`: This describes the condition that this provider must meet.  The provided idToken's `isadmin` field MUST be set to true
  * `attribute.aud=='cicp-oidc-test'`:  This describes the audience value in the token must be set to the project you are using (in my case `cicp-oidc-test`)

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
 --member "principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/subject/alice@domain.com" \
 --role roles/storage.objectAdmin
```
 
  Notice that in this mode we are **directly** allowing the federated identity access to a GCS resource

* Allow Impersonation

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
    --member "principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/subject/alice@domain.com"
```

  Then allow this service account access to GCS
```bash
gsutil iam ch serviceAccount:oidc-federated@$PROJECT_ID.iam.gserviceaccount.com:objectViewer gs://$PROJECT_ID-test
```


### Use OIDC Token

At this point, we are ready to use the OIDC token and exchange it.

Edit `main.go` and specify the variables hardcoded including the id_token from the provider earlier (yes, i'm lazy!)
```golang
	sourceToken := "eyJhbGciOiJSUzI1NiIsImtp..."
	scope := "https://www.googleapis.com/auth/cloud-platform"
	targetResource := "//iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/providers/oidc-provider-1"
	targetServiceAccount := "oidc-federated@cicp-oidc-test.iam.gserviceaccount.com"
	gcpBucketName := "cicp-oidc-test-test"
	gcpObjectName := "foo.txt"

	oTokenSource, err := sal.OIDCFederatedTokenSource(
		&sal.OIDCFederatedTokenConfig{
			SourceToken:          sourceToken,
			Scope:                scope,
			TargetResource:       targetResource,
			TargetServiceAccount: targetServiceAccount,
			UseIAMToken:          false,
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


Change the value of `UseIAMToken` to true and try running it again.  That flag will either use the federated token directly to access a resource or attempt to exchange it for an IAMCredentials token.  If set to false, the TokenSource will use **directly** use the federated token to access GCS.  If set to true, the tokensource will run the exchange


### Logging

If you used the STS token directly, the principal will appear in the GCS logs if you enabled audit logging

![images/audit_log_config.png](images/audit_log_config.png)

![images/gcs_sts_access.png](images/gcs_sts_access.png)

If you used IAM impersonation, you will see the principal performing the impersonation

![images/iam_impersonation.png](images/iam_impersonation.png)

and then the impersonated account accessing GCS

![images/gcs_iam_access.png](images/gcs_iam_access.png)

Notice the `protoPayload.authenticationInfo` structure between the two types of auth

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
   --organization=673208786092 https://securetoken.google.com/cicp-oidc-test/

      constraint: constraints/iam.workloadIdentityPoolProviders
      etag: BwWybJWeyeU=
      listPolicy:
        allowedValues:
        - https://securetoken.google.com/cicp-oidc-test/
      updateTime: '2020-10-24T15:45:19.794Z'


$ gcloud beta iam workload-identity-pools providers create-oidc oidc-provider-3 \
    --workload-identity-pool="oidc-pool-1" \
    --issuer-uri="https://securetoken.google.com/foo/" \
    --location="global" \
    --attribute-mapping="google.subject=assertion.sub,attribute.isadmin=assertion.isadmin,attribute.aud=assertion.aud" \
    --attribute-condition="attribute.isadmin=='true' && attribute.aud=='cicp-oidc-test'"
    
    ERROR: (gcloud.beta.iam.workload-identity-pools.providers.create-oidc) FAILED_PRECONDITION: Precondition check failed.
    - '@type': type.googleapis.com/google.rpc.PreconditionFailure
      violations:
      - description: "Org Policy violated for value: 'https://securetoken.google.com/foo/'."
        subject: orgpolicy:projects/user2project2/locations/global/workloadIdentityPools/oidc-pool-1
        type: constraints/iam.workloadIdentityPoolProviders

```

---

#### Create OIDC token using Identity Platform

The following shows how to acquire an OIDC token for use with this tutorial.  As mentioned, we are using FirebaseAuth/Identity Platform; you can use any other provider as long as the `.well-known` endpoint is discoverable by GCP

1. Enable Identity Platform

2. Add `Email/Password` as the provider

3. Note the `API_KEY` and authDomain value
 ![images/cicp_config.png](images/cicp_config.png)

4. Edit `login.js` and enter in the API Key/AuthDomain
  In my case, it is:

```javascript
var firebaseConfig = {
  apiKey: "AIzaSyAf7wesN7auBeyfJQJs5d_QfT24kMH7OG8",
  authDomain: "cicp-oidc-test.firebaseapp.com",
  projectId: "cicp-oidc-test",
  appId: "cicp-oidc-test",
};
```

5. Create Firebase Service Account

  - [Set up a Firebase project and service account](https://firebase.google.com/docs/admin/setup#set-up-project-and-service-account)

   Generate a service account and download it from the firebase console: (note, replace with your projectID in the URL field below)
   - [https://console.firebase.google.com/u/0/project/cicp-oidc-test/settings/serviceaccounts/adminsdk](https://console.firebase.google.com/u/0/project/cicp-oidc-test/settings/serviceaccounts/adminsdk)

   Save the file as `/tmp/svc_account.json`

   ![images/firebase_sa.png](images/firebase_sa.png)

6. Create user

```bash
npm i firebase firebase-admin
```

```bash
$ export GOOGLE_APPLICATION_CREDENTIALS=/tmp/svc_account.json

$ node create.js 

    { uid: 'alice@domain.com',
      email: 'alice@domain.com',
      emailVerified: true,
      displayName: 'alice',
      photoURL: undefined,
      phoneNumber: undefined,
      disabled: false,
      metadata:
      { lastSignInTime: null,
        creationTime: 'Sun, 25 Oct 2020 11:11:14 GMT' },
      passwordHash: undefined,
      passwordSalt: undefined,
      customClaims: { isadmin: 'true' },
      tokensValidAfterTime: 'Sun, 25 Oct 2020 11:11:14 GMT',
      tenantId: undefined,
      providerData:
      [ { uid: 'alice@domain.com',
          displayName: 'alice',
          email: 'alice@domain.com',
          photoURL: undefined,
          providerId: 'password',
          phoneNumber: undefined } ] }
```

At this pont, user Alice has a custom claim associated with the user.  Empirically, the attribute values must be string (i.e, i intentionally set `isadmin` to (string) `true` (not boolean))

7. Create id_token
  Login as that user using email/password.

  The following script actually performs a login and displays the JSON response a firebase/identity platform user would see (i.,e they would see that struct after logging in the browser too)

```json
$ node login.js 
    {
      "user": {
        "uid": "alice@domain.com",
        "displayName": "alice",
        "photoURL": null,
        "email": "alice@domain.com",
        "emailVerified": true,
        "phoneNumber": null,
        "isAnonymous": false,
        "tenantId": null,
        "providerData": [
          {
            "uid": "alice@domain.com",
            "displayName": "alice",
            "photoURL": null,
            "email": "alice@domain.com",
            "phoneNumber": null,
            "providerId": "password"
          }
        ],
        "apiKey": "AIzaSyAf7wesN7auBeyfJQJs5d_QfT24kMH7OG8",
        "appName": "[DEFAULT]",
        "authDomain": "cicp-oidc-test.firebaseapp.com",
        "stsTokenManager": {
          "apiKey": "AIzaSyAf7wesN7auBeyfJQJs5d_QfT24kMH7OG8",
          "refreshToken": "AG8BCncodfNZo5RjfUIaayD-redacted",
          "accessToken": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImQxMGM4ZjhiMGRjN2Y1NWUyYjM1NDFmMjllNWFjMzc0M2Y3N2NjZWUiLCJ0eXAiOiJKV1QifQ.eyJuYW1lIjoiYWxpY2UiLCJpc2FkbWluIjoidHJ1ZSIsImlzcyI6Imh0dHBzOi8vc2VjdXJldG9rZW4uZ29vZ2xlLmNvbS9jaWNwLW9pZGMtdGVzdCIsImF1ZCI6ImNpY3Atb2lkYy10ZXN0IiwiYXV0aF90aW1lIjoxNjAzNjI0MzAxLCJ1c2VyX2lkIjoiYWxpY2VAZG9tYWluLmNvbSIsInN1YiI6ImFsaWNlQGRvbWFpbi5jb20iLCJpYXQiOjE2MDM2MjQzMDEsImV4cCI6MTYwMzYyNzkwMSwiZW1haWwiOiJhbGljZUBkb21haW4uY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImZpcmViYXNlIjp7ImlkZW50aXRpZXMiOnsiZW1haWwiOlsiYWxpY2VAZG9tYWluLmNvbSJdfSwic2lnbl9pbl9wcm92aWRlciI6InBhc3N3b3JkIn19.oSB2vYLo8gX_CakDaO9MGHYeXGwHUySYYPhhFqL7Fx-glSrQx5O_fMSLqF0p48SvHlN47bNDYfhuwR5HRbxnn_w6XxP0cFkGInRiZngwQyFapiEbpnlT7GCU-u2KWfci0mi770giOBn4ZmiavqtmENZPyR2FcwKCRn9tPNpzFPLXG-uUPjd1zj3YblFsHwBtZo8jcmkDMMo_-Y52z5JQiHyG5sfANjldlgabnygUtInAHNvjJXDiRP0p0u4yuOjjq8mjMX9IPN1KXyHoSqaBjQCVmQqbzlx7jIl75dUxAI7x-OZ-4eZ4fWZvItYaLoQpBHQWpxLszqCYztCKz4dzxg",
          "expirationTime": 1603627901000
        },
        "redirectEventId": null,
        "lastLoginAt": "1603624301973",
        "createdAt": "1603624274304",
        "multiFactor": {
          "enrolledFactors": []
        }
      },
      "credential": null,
      "additionalUserInfo": {
        "providerId": "password",
        "isNewUser": false
      },
      "operationType": "signIn"
    }
```

Notice the `access_token` (which is actually a JWT id_token which you can decode at [jwt.io](jwt.io))

```json
  {
    "name": "alice",
    "isadmin": "true",
    "iss": "https://securetoken.google.com/cicp-oidc-test",
    "aud": "cicp-oidc-test",
    "auth_time": 1603624301,
    "user_id": "alice@domain.com",
    "sub": "alice@domain.com",
    "iat": 1603624301,
    "exp": 1603627901,
    "email": "alice@domain.com",
    "email_verified": true,
    "firebase": {
      "identities": {
        "email": [
          "alice@domain.com"
        ]
      },
      "sign_in_provider": "password"
    }
  }
```