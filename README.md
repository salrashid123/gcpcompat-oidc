## GCP Workload Identity Federation using OIDC Credentials


This is a sample procedure that will exchange an arbitrary OIDC `id_token` for a GCP credential.

You can use the GCP credential then to access any service the mapped principal has GCP IAM permissions on.

This article and repo is the second part that explores how to use the workload identity federation capability of GCP which allows for external principals (AWS,Azure or arbitrary OIDC provider) to map to a GCP credential.

The two variations described in this repo will acquire a Google Credential as described here:
 - [https://cloud.google.com/iam/docs/access-resources-oidc#generate](https://cloud.google.com/iam/docs/access-resources-oidc#generate)

The "Automatic" way is recommended and is supported by Google

The "Manual" way is also covered in this repo but I decided to wrap the steps for that into my own library here [github.com/salrashid123/oauth2/google](https://github.com/salrashid123/oauth2#usage-federated-oidc) which surfaces the credential as an [oauth2.TokenSource](https://godoc.org/golang.org/x/oauth2#TokenSource) for use in any GCP cloud library.  

>> NOTE: the library i'm using for the "manual" way is just there as an unsupported demo of a wrapped oauth2 TokenSource! 

You can certainly use either procedure but the Automatic way is included with the *supported, standard* GCP Client library.

>> This repository is not supported by Google
>> `salrashid123/oauth2/google` is also not supported by Google

also see
- [GCP Workload Identity Federation using OIDC Credentials](https://github.com/salrashid123/gcpcompat-oidc)
- [GCP Workload Identity Federation using SAML Credentials](https://github.com/salrashid123/gcpcompat-saml)
- [GCP Workload Identity Federation using AWS Credentials](https://github.com/salrashid123/gcpcompat-aws)

---

### Workload Federation - OIDC

GCP now surfaces a `STS Service` that will exchange one set of tokens for another using the GCP Secure Token Service (STS) [here](https://cloud.google.com/iam/docs/reference/sts/rest/v1beta/TopLevel/token).  These initial tokens can be either 3rd party OIDC, AWS, Azure or google `access_tokens` that are [downscoped](https://github.com/salrashid123/downscoped_token) (i.,e attenuated in permission set).

To use this tutorial, you need a GCP project with Firebase as the OIDC Provider and the ability to create user/service accounts.

Again, the two types of flows this repo demonstrates:  

- Manual Exchange:
  In this you manually do all the steps of exchanging a Firebase/Identity Platform `id_token` for a federated token and then finally use that token

- Automatic Exchange
  In this you use the google cloud client libraries to do all the heavy lifting.  << This is the recommended approach

>> It is recommended to do the manual first just to understand this capability and then move onto the automatic

#### OIDC --> GCP Identity --> GCP Resource

This tutorial will cover how to use an OIDC token and its claims to a GCP  [principal:// and principalSet://](https://cloud.google.com/iam/docs/workload-identity-federation#impersonation)


* User:  `principal://`  This maps a unique user identified by OIDC to a GCP identity

* Group: `principalSet://` This maps any user that has a given attrubute or group declared in the OIDC token to a GCP identity 


In both cases, the GCP Identity is a Service Account that the external user impersonates.

### Configure OIDC Provider

First we need an OIDC token provider that will give us an `id_token`.  Just for demonstration, we will use [Google Cloud Identity Platform](https://cloud.google.com/identity-platform) as the provider (you can of course use okta, auth0, even google itself).

The GCP project i am using in the example here is called `fabled-ray-104117`.  Identity platform will automatically create a 'bare bones' oidc `.well-known` endpoint at a url that includes the projectID:

* [https://securetoken.google.com/mineral-minutia-820/.well-known/openid-configuration](https://securetoken.google.com/mineral-minutia-820/.well-known/openid-configuration)



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
  apiKey: "AIzaSyAf7wesN7auBeyfJQJ--redacted",
  authDomain: "fabled-ray-104117.firebaseapp.com",
  projectId: "fabled-ray-104117",
  appId: "fabled-ray-104117",
};
```

5. Create Firebase Service Account

  - [Set up a Firebase project and service account](https://firebase.google.com/docs/admin/setup#set-up-project-and-service-account)

   Generate a service account and download it from the firebase console: (note, replace with your projectID in the URL field below)
   - [https://console.firebase.google.com/u/0/project/fabled-ray-104117/settings/serviceaccounts/adminsdk](https://console.firebase.google.com/u/0/project/cicp-oidc-test/settings/serviceaccounts/adminsdk)

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
        creationTime: 'Mon, 26 Jul 2021 18:10:50 GMT' },
      passwordHash: undefined,
      passwordSalt: undefined,
      customClaims: { isadmin: 'true', mygroups: [ 'group1', 'group2' ] },
      tokensValidAfterTime: 'Mon, 26 Jul 2021 18:10:50 GMT',
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
        "authDomain": "fabled-ray-104117.firebaseapp.com",
        "stsTokenManager": {
          "apiKey": "AIzaSyAf7wesN7auBeyfJQJs5d_QfT24kMH7OG8",
          "refreshToken": "AG8BCncodfNZo5RjfUIaayD-redacted",
          "accessToken": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjFiYjk2MDVjMzZlOThlMzAxMTdhNjk1MTc1NjkzODY4MzAyMDJiMmQiLCJ0eXAiOiJKV1QifQ.eyJuYW1lIjoiYWxpY2UiLCJpc2FkbWluIjoidHJ1ZSIsIm15Z3JvdXBzIjpbImdyb3VwMSIsImdyb3VwMiJdLCJpc3MiOiJodHRwczovL3NlY3VyZXRva2VuLmdvb2dsZS5jb20vZmFibGVkLXJheS0xMDQxMTciLCJhdWQiOiJmYWJsZWQtcmF5LTEwNDExNyIsImF1dGhfdGltZSI6MTYyNzMyMzEwOSwidXNlcl9pZCI6ImFsaWNlQGRvbWFpbi5jb20iLCJzdWIiOiJhbGljZUBkb21haW4uY29tIiwiaWF0IjoxNjI3MzIzMTA5LCJleHAiOjE2MjczMjY3MDksImVtYWlsIjoiYWxpY2VAZG9tYWluLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJmaXJlYmFzZSI6eyJpZGVudGl0aWVzIjp7ImVtYWlsIjpbImFsaWNlQGRvbWFpbi5jb20iXX0sInNpZ25faW5fcHJvdmlkZXIiOiJwYXNzd29yZCJ9fQ.lEJZ-tHRCOAR0kKiN9AoAB6Y7HXoSeQDQnDW2QqA1ouJd_nudlcjMW4tjsdpowtAmHprytPRsrJ35zz8r5BXu_jDX5k-aASIUw4Eg9mk1eCxXGIZn6Pv54K7MBMq-2lgzaTkpvNjX9-3jlUagOd6U0cbeTHFk4_9yA4jAS0dp2sixx_dQer0qRXJ96EIZ4XsGh_6to4DdG4n6k0V25vNchKd0vuM5G14yb2lpxmu9yE0EEXrujzgrVErFOzLZd4xeBR6svn6X-9NmYf3ffoyQv_ZF0bCTligQsBApg2aa95guig8jA5Gsf-XnuPWmSTcpu1PG54Zzw78enylr3WyHw",
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


```bash
export OIDC_TOKEN=`node login.js  | jq -r '.user.stsTokenManager.accessToken'`
echo $OIDC_TOKEN > /tmp/oidccred.txt
```

The `access_token` is actually a JWT id_token which you can decode at [jwt.io](jwt.io):

Notice the `isadmin` and `sub` fields there

```json
    {
      "name": "alice",
      "isadmin": "true",
      "mygroups": [
        "group1",
        "group2"
      ],
      "iss": "https://securetoken.google.com/fabled-ray-104117",
      "aud": "fabled-ray-104117",
      "auth_time": 1627323109,
      "user_id": "alice@domain.com",
      "sub": "alice@domain.com",
      "iat": 1627323109,
      "exp": 1627326709,
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

* `issuer` is `https://securetoken.google.com/mineral-minutia-820`,
* `sub` field describes the username
* `isadmin` is a custom claim which we will use for the `principalSet://` **attribute mapping**
* `mygroups.*` is a custom claim which we will use for the `principalSet://` **group mapping**

### Configure OIDC Federation

We can now configure the GCP project for OIDC Federation

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
    --issuer-uri="https://securetoken.google.com/fabled-ray-104117/" \
    --location="global" \
    --attribute-mapping="google.subject=assertion.sub,attribute.isadmin=assertion.isadmin,attribute.aud=assertion.aud" \
    --attribute-condition="attribute.isadmin=='true' && attribute.aud=='fabled-ray-104117'" --project $PROJECT_ID
```

  Notice the attribute mapping:
  * `google.subject=assertion.sub`:  This will extract and populate the google subject value from the provided id_token's `sub`  field.
  * `attribute.isadmin=assertion.isadmin`:  This will extract the value of the custom claim `isadmin` and then make it available for IAM rule later as an assertion

  Notice  the attribute conditions:
  * `attribute.isadmin=='true'`: This describes the condition that this provider must meet.  The provided idToken's `isadmin` field MUST be set to true
  * `attribute.aud=='fabled-ray-104117'`:  This describes the audience value in the token must be set to the project you are using (in my case `fabled-ray-104117`)

If you set the attribute conditions to something else, you should see an error during authentication:

```
Unable to exchange token {"error":"unauthorized_client","error_description":"The given credential is rejected by the attribute condition."},
```

For group mapping

```bash
gcloud beta iam workload-identity-pools providers create-oidc oidc-provider-2 \
    --workload-identity-pool="oidc-pool-1" \
    --issuer-uri="https://securetoken.google.com/fabled-ray-104117/" \
    --location="global" \
    --attribute-mapping="google.subject=assertion.sub,google.groups=assertion.mygroups,attribute.aud=assertion.aud" \
    --attribute-condition="attribute.aud=='fabled-ray-104117'" --project $PROJECT_ID
```

* Create GCS Resource

  Create a test GCP resource like GCS and upload a file

```bash
gsutil mb gs://$PROJECT_ID-test
echo fooooo > foo.txt
gsutil cp foo.txt gs://$PROJECT_ID-test
```

* Crate Service Account

  By default, the OIDC identity will need to map to a Service Account which inturn will have access to the GCS resource.  The external identity will get validated by GCP and then will impersonate a GCP Service Account

```bash
gcloud iam service-accounts create oidc-federated
```

* Allow federated identity map

This allows a single user `alice@domain.com` permissions to impersonate the Service Account

```bash

gcloud iam service-accounts add-iam-policy-binding oidc-federated@$PROJECT_ID.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/subject/alice@domain.com"
```

- A) Attribute Mapping

This allows any user part of the OIDC issuer that has the claim embedded in the token with key-value `isadmin==true`

```bash
gcloud iam service-accounts add-iam-policy-binding oidc-federated@$PROJECT_ID.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/attribute.isadmin/true"
```

- B) Group Mapping

The following allows any user that is in `group` of the claim permission to impersonate this service account

```bash
gcloud iam service-accounts add-iam-policy-binding oidc-federated@$PROJECT_ID.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/group/group1"
```

* Allow service account access to GCS

```bash
gsutil iam ch serviceAccount:oidc-federated@$PROJECT_ID.iam.gserviceaccount.com:objectViewer gs://$PROJECT_ID-test
```

### Manual


Before you run the sample, you must first get an OIDC Token.  See `Create OIDC token using Identity Platform` above

```bash
export OIDC_TOKEN=`node login.js  | jq -r '.user.stsTokenManager.accessToken'`
echo $OIDC_TOKEN > /tmp/oidccred.txt
```

At this point, we are ready to use the OIDC token and exchange it manually

- Attribute Mapping 

```bash
$ go run main.go \
   --gcpBucket $PROJECT_ID-test \
   --gcpObjectName foo.txt \
   --gcpResource //iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/providers/oidc-provider-1 \
   --gcpTargetServiceAccount oidc-federated@$PROJECT_ID.iam.gserviceaccount.com \
   --useIAMToken \
   --sourceToken $OIDC_TOKEN

2020/10/24 07:16:14 OIDC Derived GCP access_token: ya29.c.KuQC4gf-xkKbOCIzRGAmAPdL2unF4vLCjZG7TZv7l7bjCK67n2qduIFDs63HR...

fooooo
```

- Group Mapping 

```bash
$ go run main.go \
   --gcpBucket $PROJECT_ID-test \
   --gcpObjectName foo.txt \
   --gcpResource //iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/providers/oidc-provider-2 \
   --gcpTargetServiceAccount oidc-federated@$PROJECT_ID.iam.gserviceaccount.com \
   --useIAMToken \
   --sourceToken $OIDC_TOKEN

2020/10/24 07:16:14 OIDC Derived GCP access_token: ya29.c.KuQC4gf-xkKbOCIzRGAmAPdL2unF4vLCjZG7TZv7l7bjCK67n2qduIFDs63HR...

fooooo
```

What you should see is the output of the GCS file


### Automatic

We are now ready to use the Automatic Application Default Credentials to access the ressource

First configure the ADC bootstrap file:

```bash
gcloud beta iam workload-identity-pools create-cred-config  \
  projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/providers/oidc-provider-1   \
  --service-account=oidc-federated@$PROJECT_ID.iam.gserviceaccount.com   \
  --output-file=sts-creds.json  \
  --credential-source-file=/tmp/oidccred.txt
```

The output/bootstrap file should look something like this:

```json
{
  "type": "external_account",
  "audience": "//iam.googleapis.com/projects/1071284184436/locations/global/workloadIdentityPools/oidc-pool-1/providers/oidc-provider-1",
  "subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
  "token_url": "https://sts.googleapis.com/v1/token",
  "credential_source": {
    "file": "/tmp/oidccred.txt"
  },
  "service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/oidc-federated@cicp-oidc-test.iam.gserviceaccount.com:generateAccessToken"
}
```

Notice the bootstrap file has a pointer to the file where the actual creds exist `/tmp/oidccreds.txt`.  (eventually other cred sources should be supported)

Before you run the sample, you must first get an OIDC Token.  See `Create OIDC token using Identity Platform` below

> Remember `/tmp/oidccred.txt` has the content of the raw OIDC token to use (i.,e is the value of $OIDC_TOKEN)

```bash
export OIDC_TOKEN=`node login.js  | jq -r '.user.stsTokenManager.accessToken'`
echo $OIDC_TOKEN > /tmp/oidccred.txt
```

```bash
export GOOGLE_APPLICATION_CREDENTIALS=`pwd`/sts-creds.json

go run main.go    --gcpBucket $PROJECT_ID-test    --gcpObjectName foo.txt --useADC 
```

> If you would rather use `google.group=` mapping wiht a `principalSet://`, change the `create-cred-config` to use `oidc-provider-2`


At the time of writing (3/14/21), the configuration file (only supports reading of a file that contains the oidc token (`/tmp/oidccred.txt`) directly.  Eventually, other mechanisms like url or executing a binary that returns the oidc token will be supported

### Using Federated or IAM Tokens

GCP STS Tokens can be used directly against a few GCP services as described here

Skip step `(5)` of [Exchange Token](https://cloud.google.com/iam/docs/access-resources-oidc#exchange-token)

What that means is you can skip the step to exchange the GCP Federation token for an Service Account token and _directly_ apply IAM policies on the resource.

This not only saves the step of running the exchange but omits the need for a secondary GCP service account to impersonate.

To use GCS, allow either the mapped identity direct access to the resource.  In this case `storage.objectAdmin` access which we already allowed earlier:


* Allow Federated Identity IAM access

  Configure the federated identity access to GCS bucket. 
This allows a single user `alice@domain.com` 

```bash
gcloud projects add-iam-policy-binding $PROJECT_ID  \
 --member "principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/subject/alice@domain.com" \
 --role roles/storage.objectAdmin
```

```bash
gcloud projects add-iam-policy-binding $PROJECT_ID  \
 --member "principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/attribute.isadmin/true" \
 --role roles/storage.objectAdmin
```

```bash
gcloud projects add-iam-policy-binding $PROJECT_ID  \
 --member "principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/oidc-pool-1/group/group1" \
 --role roles/storage.objectAdmin
```
 
  Notice that in this mode we are **directly** allowing the federated identity access to a GCS resource


To use Federated tokens, use remove the `--useIAMToken` flag


>> **If** you want to use Federated tokens only with the Automatic flow, delete `service_account_impersonation_url` declaration in `sts-creds.json`


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
