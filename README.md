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
RUN_SERVICE="go-modproxy"
REGION="europe-west4"
BUILD_REGION="europe-west1"
ARTIFACTS_REPOSITORY="cloud-run-source-deploy"
WORKLOAD_IDENTITY_PROVIDER_NAME="go-modproxy"
GITHUB_REPOSITORY="epiccoolguy/go-modproxy"
RUN_SERVICE_ACCOUNT_NAME="run-${RUN_SERVICE}"
GOOGLE_CLOUD_RUN_SERVICE_ACCOUNT="${RUN_SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

PROJECT_ID=$(head /dev/urandom | LC_ALL=C tr -dc 0-9 | head -c4 | sed -e "s/^/${PROJECT_PREFIX}-/" | cut -c 1-30)
CLOUDBUILD_BUCKET="gs://${PROJECT_ID}_cloudbuild"
REPOSITORY_URI=${REGION}-docker.pkg.dev/${PROJECT_ID}/${ARTIFACTS_REPOSITORY}/${RUN_SERVICE}
BILLING_ACCOUNT_ID=$(gcloud billing accounts list --filter="OPEN = True" --format="value(ACCOUNT_ID)")

# Add variables for the CD workflow to Github Repository Variables
echo "${RUN_SERVICE}" | gh variable set GOOGLE_SERVICE_NAME --repo="epiccoolguy/go-modproxy"
echo "${REGION}" | gh variable set GOOGLE_SERVICE_REGION --repo="epiccoolguy/go-modproxy"
echo "${PROJECT_ID}" | gh variable set GOOGLE_PROJECT_ID --repo="epiccoolguy/go-modproxy"
echo "${HOST_PATTERN}" | gh variable set HOST_PATTERN --repo="epiccoolguy/go-modproxy"
echo "${HOST_REPLACEMENT}" | gh variable set HOST_REPLACEMENT --repo="epiccoolguy/go-modproxy"
echo "${PATH_PATTERN}" | gh variable set PATH_PATTERN --repo="epiccoolguy/go-modproxy"
echo "${PATH_REPLACEMENT}" | gh variable set PATH_REPLACEMENT --repo="epiccoolguy/go-modproxy"

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

# Create an intermediate service account for the workload identity pool to impersonate.
gcloud iam service-accounts create "${RUN_SERVICE_ACCOUNT_NAME}" --project "${PROJECT_ID}"

# Add the intermediate service account as a Github repository secret for google-github-actions/auth@v2:
echo "${GOOGLE_CLOUD_RUN_SERVICE_ACCOUNT}" | gh secret set GOOGLE_CLOUD_RUN_SERVICE_ACCOUNT --repo="epiccoolguy/go-modproxy"

# Create a Workload Identity Pool
gcloud iam workload-identity-pools create "githubactions" \
  --display-name="GitHub Actions Pool" \
  --location="global" \
  --project="${PROJECT_ID}"

# Get the full id of the pool
WORKLOAD_IDENTITY_POOL_ID=$(gcloud iam workload-identity-pools describe "githubactions" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --format="value(name)")

# Create a Workload Identity Provider within the pool
gcloud iam workload-identity-pools providers create-oidc "${WORKLOAD_IDENTITY_PROVIDER_NAME}" \
  --workload-identity-pool="githubactions" \
  --display-name="${GITHUB_REPOSITORY} provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --location="global" \
  --project="${PROJECT_ID}"

# Get the full id of the provider
WORKLOAD_IDENTITY_PROVIDER_ID=$(gcloud iam workload-identity-pools providers describe "${WORKLOAD_IDENTITY_PROVIDER_NAME}" \
  --workload-identity-pool="githubactions" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --format="value(name)")

# Add the identity provider as a Github repository secret for google-github-actions/auth@v2
echo "${WORKLOAD_IDENTITY_PROVIDER_ID}" | gh secret set GOOGLE_WORKLOAD_IDENTITY_PROVIDER_ID --repo="epiccoolguy/go-modproxy"

# Grant the Github Actions Pool access to the intermediate service account in a specific repository
PRINCIPAL="principalSet://iam.googleapis.com/${WORKLOAD_IDENTITY_POOL_ID}/attribute.repository/${GITHUB_REPOSITORY}"
gcloud iam service-accounts add-iam-policy-binding \
  "${GOOGLE_CLOUD_RUN_SERVICE_ACCOUNT}" \
  --member="${PRINCIPAL}" \
  --role="roles/iam.workloadIdentityUser" \
  --project="${PROJECT_ID}"

# Grant the intermediate service account access to Cloud Run within the project
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${GOOGLE_CLOUD_RUN_SERVICE_ACCOUNT}" \
  --role="roles/run.developer"
```

Enable unauthenticated invocations in an organisation enforcing DRS using Resource Manager tags and a conditional DRS policy:

```sh
PROJECT_NUMBER=$(gcloud projects describe "${PROJECT_ID}" --format="value(projectNumber)")
ORGANIZATION_ID=$(gcloud projects describe "${PROJECT_ID}" --format="value(parent.id)")

gcloud resource-manager tags bindings create \
  --tag-value="${ORGANIZATION_ID}/allUsersIngress/True" \
  --parent="//run.googleapis.com/projects/${PROJECT_NUMBER}/locations/${REGION}/services/${RUN_SERVICE}" \
  --location=${REGION}

# This can fail until the binding has propagated
gcloud run services add-iam-policy-binding "${RUN_SERVICE}" \
  --member="allUsers" \
  --role="roles/run.invoker" \
  --region="${REGION}" \
  --project="${PROJECT_ID}"
```

---

Map the Cloud Run instance to a custom domain:

```sh
DOMAIN="go.loafoe.dev"

# It can take up to 30 minutes for Cloud Run to issue provision a certificate and route
gcloud beta run domain-mappings create --service="${RUN_SERVICE}" --domain="${DOMAIN}" --region="${REGION}" --project="${PROJECT_ID}"
```

Retrieve the necessary DNS record information for the domain mappings:

```sh
gcloud beta run domain-mappings describe --domain="${DOMAIN}" --region="${REGION}" --project="${PROJECT_ID}"
```
