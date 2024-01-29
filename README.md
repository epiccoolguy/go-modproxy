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
SERVICE="go-modproxy"
REGION="europe-west4"
BUILD_REGION="europe-west1" # https://cloud.google.com/build/docs/locations#restricted_regions_for_some_projects
BILLING_ACCOUNT_ID=$(gcloud billing accounts list --filter="OPEN = True" --format="value(ACCOUNT_ID)") # use first enabled billing account
PROJECT_ID="go-modproxy"
CLOUDBUILD_BUCKET="gs://${PROJECT_ID}_cloudbuild"
ARTIFACTS_REPOSITORY="cloud-run-source-deploy"
REPOSITORY_URI=$REGION-docker.pkg.dev/$PROJECT_ID/$ARTIFACTS_REPOSITORY/$SERVICE
HOST_PATTERN="go.loafoe.dev"
HOST_REPLACEMENT="github.com"
PATH_PATTERN="/"
PATH_REPLACEMENT="/epiccoolguy/go-"

# Create new GCP project
gcloud projects create "$PROJECT_ID"

# Link billing account
gcloud billing projects link "$PROJECT_ID" --billing-account="$BILLING_ACCOUNT_ID"

# Enable required GCP services
gcloud services enable artifactregistry.googleapis.com cloudbuild.googleapis.com run.googleapis.com --project="$PROJECT_ID"

# Create GCS bucket for builds
gcloud storage buckets create "$CLOUDBUILD_BUCKET" --location="$REGION" --project="$PROJECT_ID"

# Create Docker artifact repository
gcloud artifacts repositories create "$ARTIFACTS_REPOSITORY" --repository-format=docker --location="$REGION" --project="$PROJECT_ID"

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
