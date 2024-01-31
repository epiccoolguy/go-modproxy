#!/bin/sh

# Application variables
HOST_PATTERN="go.loafoe.dev"
HOST_REPLACEMENT="github.com"
PATH_PATTERN="/"
PATH_REPLACEMENT="/epiccoolguy/go-"

# Github variables
GH_REPOSITORY="epiccoolguy/go-modproxy"

# Google Cloud Platform (GCP) project variables
GCP_PROJECT_NAME="Go Module Proxy"
GCP_PROJECT_PREFIX="go-modproxy"
GCP_PROJECT_REGION="europe-west4"
GCP_PROJECT_ID=$(head /dev/urandom | LC_ALL=C tr -dc 0-9 | head -c6 | sed -e "s/^/${GCP_PROJECT_PREFIX}-/" | cut -c 1-30)

# Google Artifact Registry (GAR) variables
GAR_REPOSITORY="cloud-run-source-deploy"
GAR_LOCATION="${GCP_PROJECT_REGION}"
GAR_IMAGE_URI=${GAR_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${GAR_REPOSITORY}/${GH_REPOSITORY}

# Google Cloud Storage (GCS) variables
GCS_CLOUDBUILD_BUCKET="gs://${GCP_PROJECT_ID}_cloudbuild"
GCS_REGION="${GCP_PROJECT_REGION}"

# Google Cloud Build (GCB) variables
GCB_REGION="europe-west1"

# Google Cloud Run (GCR) variables
GCR_SERVICE="${GCP_PROJECT_PREFIX}"
GCR_REGION="${GCP_PROJECT_REGION}"
GCR_DOMAIN="go.loafoe.dev"

# Workload Identity Federation (WIF) variables
WIF_SERVICE_ACCOUNT_NAME="${GCP_PROJECT_PREFIX}"
WIF_SERVICE_ACCOUNT="${WIF_SERVICE_ACCOUNT_NAME}@${GCP_PROJECT_ID}.iam.gserviceaccount.com"
WIF_PROVIDER_NAME="${GCP_PROJECT_PREFIX}"

# Create new GCP project
gcloud projects create "${GCP_PROJECT_ID}" --name "${GCP_PROJECT_NAME}"

# Link billing account
BILLING_ACCOUNT_ID=$(gcloud billing accounts list --filter="OPEN = True" --format="value(ACCOUNT_ID)")
gcloud billing projects link "${GCP_PROJECT_ID}" --billing-account="${BILLING_ACCOUNT_ID}"

# Enable required GCP services
gcloud services enable \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  run.googleapis.com \
  iamcredentials.googleapis.com \
  --project="${GCP_PROJECT_ID}"

# Create GCS bucket for builds
gcloud storage buckets create "${GCS_CLOUDBUILD_BUCKET}" --location="${GCS_REGION}" --project="${GCP_PROJECT_ID}"

# Create Docker artifact repository
gcloud artifacts repositories create "${GAR_REPOSITORY}" \
  --repository-format=docker \
  --location="${GAR_LOCATION}" \
  --project="${GCP_PROJECT_ID}"

# Create an intermediate service account for the workload identity pool to impersonate.
gcloud iam service-accounts create "${WIF_SERVICE_ACCOUNT_NAME}" --project "${GCP_PROJECT_ID}"

# Create a Workload Identity Pool
gcloud iam workload-identity-pools create "githubactions" \
  --display-name="GitHub Actions Pool" \
  --location="global" \
  --project="${GCP_PROJECT_ID}"

# Get the full id of the pool
WORKLOAD_IDENTITY_POOL_ID=$(gcloud iam workload-identity-pools describe "githubactions" \
  --project="${GCP_PROJECT_ID}" \
  --location="global" \
  --format="value(name)")

# Create a Workload Identity Provider within the pool
gcloud iam workload-identity-pools providers create-oidc "${WIF_PROVIDER_NAME}" \
  --workload-identity-pool="githubactions" \
  --display-name="${GH_REPOSITORY} provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --location="global" \
  --project="${GCP_PROJECT_ID}"

# Get the full id of the provider
WIF_PROVIDER_ID=$(gcloud iam workload-identity-pools providers describe "${WIF_PROVIDER_NAME}" \
  --workload-identity-pool="githubactions" \
  --project="${GCP_PROJECT_ID}" \
  --location="global" \
  --format="value(name)")

# Grant the Github Actions Pool access to the intermediate service account in a specific repository
PRINCIPAL="principalSet://iam.googleapis.com/${WORKLOAD_IDENTITY_POOL_ID}/attribute.repository/${GH_REPOSITORY}"
gcloud iam service-accounts add-iam-policy-binding \
  "${WIF_SERVICE_ACCOUNT}" \
  --member="${PRINCIPAL}" \
  --role="roles/iam.workloadIdentityUser" \
  --project="${GCP_PROJECT_ID}"

# Grant the intermediate service account access to Cloud Run within the project
gcloud projects add-iam-policy-binding "${GCP_PROJECT_ID}" --member="serviceAccount:${WIF_SERVICE_ACCOUNT}" --role="roles/run.developer"

# Grant the intermediate service account access to Artifact Registry within the project
gcloud artifacts repositories add-iam-policy-binding "${GAR_REPOSITORY}" \
  --member="serviceAccount:${WIF_SERVICE_ACCOUNT}" \
  --role="roles/artifactregistry.writer" \
  --location="${GAR_LOCATION}" \
  --project="${GCP_PROJECT_ID}"

# Grant the intermediate service account permission to "ActAs" itself to be able to deploy to Cloud Run.
gcloud iam service-accounts add-iam-policy-binding "${WIF_SERVICE_ACCOUNT}" \
  --member="serviceAccount:${WIF_SERVICE_ACCOUNT}" \
  --role="roles/iam.serviceAccountUser" \
  --project="${GCP_PROJECT_ID}"

# Submit build to Cloud Build
COMMIT_SHA=$(git rev-parse HEAD)
gcloud builds submit . \
  --config cloudbuild.yaml \
  --substitutions="_REPOSITORY_URI=${GAR_IMAGE_URI},COMMIT_SHA=${COMMIT_SHA}" \
  --region="${GCB_REGION}" \
  --project="${GCP_PROJECT_ID}"

# Deploy service to Cloud Run
gcloud run deploy "${GCR_SERVICE}" \
  --image="${GAR_IMAGE_URI}:${COMMIT_SHA}" \
  --set-env-vars="HOST_PATTERN=${HOST_PATTERN},HOST_REPLACEMENT=${HOST_REPLACEMENT},PATH_PATTERN=${PATH_PATTERN},PATH_REPLACEMENT=${PATH_REPLACEMENT}" \
  --no-allow-unauthenticated \
  --service-account="${WIF_SERVICE_ACCOUNT}" \
  --region="${GCR_REGION}" \
  --project="${GCP_PROJECT_ID}"

# Enable unauthenticated invocations in an organisation enforcing DRS using Resource Manager tags and a conditional DRS policy:
PROJECT_NUMBER=$(gcloud projects describe "${GCP_PROJECT_ID}" --format="value(projectNumber)")
ORGANIZATION_ID=$(gcloud projects describe "${GCP_PROJECT_ID}" --format="value(parent.id)")
gcloud resource-manager tags bindings create \
  --tag-value="${ORGANIZATION_ID}/allUsersIngress/True" \
  --parent="//run.googleapis.com/projects/${PROJECT_NUMBER}/locations/${GCR_REGION}/services/${GCR_SERVICE}" \
  --location=${GCR_REGION}

# This can fail until the binding has propagated
gcloud run services add-iam-policy-binding "${GCR_SERVICE}" \
  --member="allUsers" \
  --role="roles/run.invoker" \
  --region="${GCR_REGION}" \
  --project="${GCP_PROJECT_ID}"

# Map the Cloud Run service to a custom domain. It can take up to 30 minutes for Cloud Run to provision a certificate and start routing
gcloud beta run domain-mappings create \
  --service="${GCR_SERVICE}" \
  --domain="${GCR_DOMAIN}" \
  --region="${GCR_REGION}" \
  --project="${GCP_PROJECT_ID}"

# Retrieve the necessary DNS record information for the domain mappings:
gcloud beta run domain-mappings describe \
  --domain="${GCR_DOMAIN}" \
  --region="${GCR_REGION}" \
  --project="${GCP_PROJECT_ID}"

# Add variables for the CD workflow to Github Repository Variables
echo "${HOST_PATTERN}" | gh variable set HOST_PATTERN --repo="epiccoolguy/go-modproxy"
echo "${HOST_REPLACEMENT}" | gh variable set HOST_REPLACEMENT --repo="epiccoolguy/go-modproxy"
echo "${PATH_PATTERN}" | gh variable set PATH_PATTERN --repo="epiccoolguy/go-modproxy"
echo "${PATH_REPLACEMENT}" | gh variable set PATH_REPLACEMENT --repo="epiccoolguy/go-modproxy"
echo "${GCP_PROJECT_ID}" | gh variable set GCP_PROJECT_ID --repo="epiccoolguy/go-modproxy"
echo "${GAR_REPOSITORY}" | gh variable set GAR_REPOSITORY --repo="epiccoolguy/go-modproxy"
echo "${GAR_LOCATION}" | gh variable set GAR_LOCATION --repo="epiccoolguy/go-modproxy"
echo "${GCR_SERVICE}" | gh variable set GCR_SERVICE --repo="epiccoolguy/go-modproxy"
echo "${GCR_REGION}" | gh variable set GCR_REGION --repo="epiccoolguy/go-modproxy"
echo "${WIF_SERVICE_ACCOUNT}" | gh secret set WIF_SERVICE_ACCOUNT --repo="epiccoolguy/go-modproxy"
echo "${WIF_PROVIDER_ID}" | gh secret set WIF_PROVIDER_ID --repo="epiccoolguy/go-modproxy"
