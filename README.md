# modproxy

> modproxy directs `go get` from one location to another.

## Configuration

The behaviour of modproxy can be configured using the following environment variables:

- `HOST_PATTERN`: Specifies the pattern for host matching. Defaults to "go.loafoe.dev".
- `HOST_REPLACEMENT`: Determines the replacement for the host. Defaults to "github.com".
- `PATH_PATTERN`: Sets the pattern for path matching. Defaults to "/".
- `PATH_REPLACEMENT`: Defines the replacement for the path. Defaults to "/epiccoolguy/go-".

## Run locally

```sh
FUNCTION_TARGET=ModProxy LOCAL_ONLY=true go run cmd/main.go
```

- `FUNCTION_TARGET`: Specifies the name of the function to be executed when the server is started.
- `LOCAL_ONLY`: When set to true, the server listens only on 127.0.0.1 (localhost), restricting access to the local machine. This is useful for local testing, avoiding firewall warnings, and preventing external access to the server during development or testing phases. If not set, listen on all interfaces.

Confirm the url is correctly being rewritten:

```sh
curl -H 'Host: go.loafoe.dev' localhost:8080/modproxy
# Output: <html><head><meta name="go-import" content="go.loafoe.dev/modproxy git https://github.com/epiccoolguy/go-modproxy"></head><body></body></html>
```

## Run using `pack` and Docker

```sh
pack build \
  --builder gcr.io/buildpacks/builder:v1 \
  --env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
  --env GOOGLE_FUNCTION_TARGET=ModProxy \
  go-modproxy
```

- `GOOGLE_FUNCTION_SIGNATURE_TYPE`: Specifies the type of function signature the application uses.
- `GOOGLE_FUNCTION_TARGET`: Specifies the name of the function to execute in the application.

Run the built image:

```sh
docker run --rm -p 8080:8080 go-modproxy
```

Confirm the url is correctly being rewritten:

```sh
curl -H 'Host: go.loafoe.dev' localhost:8080/modproxy
# Output: <html><head><meta name="go-import" content="go.loafoe.dev/modproxy git https://github.com/epiccoolguy/go-modproxy"></head><body></body></html>
```

## Run using Google Cloud Platform

```sh
HOST_PATTERN="go.loafoe.dev"
HOST_REPLACEMENT="github.com"
PATH_PATTERN="/"
PATH_REPLACEMENT="/epiccoolguy/go-"

PROJECT_NAME="Go Module Proxy"
PROJECT_PREFIX="go-modproxy"
SERVICE="go-modproxy"
REGION="europe-west4"
BUILD_REGION="europe-west1"
ARTIFACTS_REPOSITORY="cloud-run-source-deploy"

PROJECT_ID=$(head /dev/urandom | LC_ALL=C tr -dc 0-9 | head -c4 | sed -e "s/^/${PROJECT_PREFIX}-/" | cut -c 1-30)
CLOUDBUILD_BUCKET="gs://${PROJECT_ID}_cloudbuild"
REPOSITORY_URI=${REGION}-docker.pkg.dev/${PROJECT_ID}/${ARTIFACTS_REPOSITORY}/${SERVICE}
BILLING_ACCOUNT_ID=$(gcloud billing accounts list --filter="OPEN = True" --format="value(ACCOUNT_ID)")

# Create new GCP project
gcloud projects create "${PROJECT_ID}" --name "${PROJECT_NAME}"

# Link billing account
gcloud billing projects link "${PROJECT_ID}" --billing-account="${BILLING_ACCOUNT_ID}"

# Enable required GCP services
gcloud services enable artifactregistry.googleapis.com cloudbuild.googleapis.com run.googleapis.com iamcredentials.googleapis.com --project="${PROJECT_ID}"

# Create GCS bucket for builds
gcloud storage buckets create "${CLOUDBUILD_BUCKET}" --location="${REGION}" --project="${PROJECT_ID}"

# Create Docker artifact repository
gcloud artifacts repositories create "${ARTIFACTS_REPOSITORY}" --repository-format=docker --location="${REGION}" --project="${PROJECT_ID}"

# Submit build to Cloud Build
gcloud builds submit . --config cloudbuild.yaml --substitutions=_REPOSITORY_URI=$REPOSITORY_URI,COMMIT_SHA=$(git rev-parse HEAD) --region="$BUILD_REGION" --project="$PROJECT_ID"

# Deploy service to Cloud Run
gcloud run deploy "$SERVICE" --image="$REPOSITORY_URI:$(git rev-parse HEAD)" --set-env-vars="HOST_PATTERN=$HOST_PATTERN,HOST_REPLACEMENT=$HOST_REPLACEMENT,PATH_PATTERN=$PATH_PATTERN,PATH_REPLACEMENT=$PATH_REPLACEMENT" --no-allow-unauthenticated --region="$REGION" --project="$PROJECT_ID"
```

Enable unauthenticated invocations in an organisation enforcing DRS using Resource Manager tags and a conditional DRS policy:

```sh
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format="value(projectNumber)")
ORGANIZATION_ID=$(gcloud projects describe "$PROJECT_ID" --format="value(parent.id)")

gcloud resource-manager tags bindings create \
  --tag-value="$ORGANIZATION_ID/allUsersIngress/True" \
  --parent="//run.googleapis.com/projects/$PROJECT_NUMBER/locations/$REGION/services/$SERVICE" \
  --location=$REGION

# This can fail until the binding has propagated
gcloud run services add-iam-policy-binding "$SERVICE" \
  --member="allUsers" \
  --role="roles/run.invoker" \
  --region="$REGION" \
  --project="$PROJECT_ID"
```

---

Map the Cloud Run instance to a custom domain:

```sh
DOMAIN="go.loafoe.dev"

# It can take up to 30 minutes for Cloud Run to issue provision a certificate and route
gcloud beta run domain-mappings create --service="$SERVICE" --domain="$DOMAIN" --region="$REGION" --project="$PROJECT_ID"
```

Retrieve the necessary DNS record information for the domain mappings:

```sh
gcloud beta run domain-mappings describe --domain="$DOMAIN" --region="$REGION" --project="$PROJECT_ID"
```

## Direct Workload Identity Federation

Set up variables:

```sh
PROJECT_ID="go-modproxy"
REPOSITORY="go-modproxy"
SERVICE="go-modproxy"
```

Create a Workload Identity Pool and get its full ID:

```sh
gcloud iam workload-identity-pools create "github" \
  --display-name="GitHub Actions Pool" \
  --location="global" \
  --project="$PROJECT_ID"

WORKLOAD_IDENTITY_POOL_ID=$(gcloud iam workload-identity-pools describe "github" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --format="value(name)")
```

Create a Workload Identity Provider within the pool and get its resource name:

```sh
gcloud iam workload-identity-pools providers create-oidc "$REPOSITORY" \
  --workload-identity-pool="github" \
  --display-name="$REPOSITORY provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --location="global" \
  --project="$PROJECT_ID"

WORKLOAD_IDENTITY_PROVIDER_ID=$(gcloud iam workload-identity-pools providers describe "$REPOSITORY" \
  --workload-identity-pool="github" \
  --project="$PROJECT_ID" \
  --location="global" \
  --format="value(name)")
```

Add the identity provider as a Github repository secret for google-github-actions/auth@v2:

```sh
echo "$WORKLOAD_IDENTITY_PROVIDER_ID" | gh secret set GOOGLE_WORKLOAD_IDENTITY_PROVIDER_ID --repo="epiccoolguy/go-modproxy"
```

Grant access to Google Cloud Run from the Github Actions Pool in a specific repository:

```sh
REPOSITORY="epiccoolguy/go-modproxy"
SERVICE="go-modproxy"
REGION="europe-west4"

gcloud run services add-iam-policy-binding "$SERVICE" \
  --member="principalSet://iam.googleapis.com/${WORKLOAD_IDENTITY_POOL_ID}/attribute.repository/${REPOSITORY}" \
  --role="roles/run.admin" \
  --region="$REGION" \
  --project="$PROJECT_ID"
```

## Setting up Github repository variables

```sh
echo "go-modproxy" | gh variable set GOOGLE_SERVICE_NAME --repo="epiccoolguy/go-modproxy"
echo "europe-west4" | gh variable set GOOGLE_SERVICE_REGION --repo="epiccoolguy/go-modproxy"
echo "go-modproxy" | gh variable set GOOGLE_PROJECT_ID --repo="epiccoolguy/go-modproxy"
echo "go.loafoe.dev" | gh variable set HOST_PATTERN --repo="epiccoolguy/go-modproxy"
echo "github.com" | gh variable set HOST_REPLACEMENT --repo="epiccoolguy/go-modproxy"
echo "/" | gh variable set PATH_PATTERN --repo="epiccoolguy/go-modproxy"
echo "/epiccoolguy/go-" | gh variable set PATH_REPLACEMENT --repo="epiccoolguy/go-modproxy"
```
