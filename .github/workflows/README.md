# CI/CD Workflows

This directory contains GitHub Actions workflows for continuous integration and deployment.

## Workflows

### 1. CI (`ci.yml`)

Runs on every push and pull request to main branches.

**Jobs:**
- **Lint**: Runs `golangci-lint` to check code quality
- **Test**: Runs unit tests with race detection and coverage
- **Build**: Builds all Go services
- **Docker Build**: Builds Docker image (cached)
- **Security**: Runs Gosec security scanner

**Triggers:**
- Push to `master`, `main`, or `develop`
- Pull requests to `master`, `main`, or `develop`

### 2. CD (`cd.yml`)

Handles continuous deployment to staging and production.

**Jobs:**
- **Build and Push**: Builds multi-arch Docker images and pushes to GitHub Container Registry
- **Deploy Staging**: Deploys to staging Kubernetes cluster (on push to main branch)
- **Deploy Production**: Deploys to production (on version tags or manual trigger)

**Triggers:**
- Push to `master` or `main` → Deploy to staging
- Push tag `v*` → Deploy to production
- Manual workflow dispatch → Choose environment

**Required Secrets:**
- `KUBECONFIG_STAGING`: Base64-encoded kubeconfig for staging
- `KUBECONFIG_PRODUCTION`: Base64-encoded kubeconfig for production

### 3. Integration Tests (`integration-tests.yml`)

Runs integration tests against real services.

**Jobs:**
- **Integration Tests**: Runs integration test suite with TimescaleDB and Redis

**Triggers:**
- Push to main branches
- Pull requests
- Manual dispatch

### 4. Docker Compose Test (`docker-compose-test.yml`)

Tests the complete system using Docker Compose.

**Jobs:**
- **Docker Compose Test**: 
  - Starts all infrastructure services
  - Runs database migrations
  - Builds and starts application services
  - Verifies all services are healthy

**Triggers:**
- Push to main branches
- Pull requests
- Manual dispatch

### 5. Release (`release.yml`)

Creates release artifacts for GitHub releases.

**Jobs:**
- **Release**: Builds binaries for multiple platforms (Linux, macOS, Windows, ARM64)

**Triggers:**
- GitHub release created
- Manual dispatch with version input

## Setup

### GitHub Secrets

Configure the following secrets in GitHub repository settings:

1. **KUBECONFIG_STAGING**: Base64-encoded kubeconfig for staging cluster
   ```bash
   cat ~/.kube/config | base64 -w 0
   ```

2. **KUBECONFIG_PRODUCTION**: Base64-encoded kubeconfig for production cluster
   ```bash
   cat ~/.kube/config | base64 -w 0
   ```

### GitHub Environments

Create environments in GitHub repository settings:

1. **staging**
   - Protection rules: Optional
   - Deployment branch: `master` or `main`
   - URL: Your staging environment URL

2. **production**
   - Protection rules: Required reviewers
   - Deployment branch: Tags matching `v*`
   - URL: Your production environment URL

### Container Registry

The workflows use GitHub Container Registry (ghcr.io). Images are automatically published to:
- `ghcr.io/<owner>/<repo>:<branch>`
- `ghcr.io/<owner>/<repo>:<sha>`
- `ghcr.io/<owner>/<repo>:<tag>` (for releases)
- `ghcr.io/<owner>/<repo>:latest` (for main branch)

## Usage

### Running Tests Locally

```bash
# Run linter
make lint

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
go test -v -tags=integration ./tests/...
```

### Building Docker Images

```bash
# Build locally
docker build -t stock-scanner:local .

# Build with BuildKit
DOCKER_BUILDKIT=1 docker build -t stock-scanner:local .
```

### Manual Deployment

1. Go to Actions → CD workflow
2. Click "Run workflow"
3. Select environment (staging/production)
4. Click "Run workflow"

### Creating a Release

1. Create a new tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. Or create a GitHub release via UI (triggers release workflow)

## Workflow Status Badges

Add to your README.md:

```markdown
![CI](https://github.com/<owner>/<repo>/workflows/CI/badge.svg)
![CD](https://github.com/<owner>/<repo>/workflows/CD/badge.svg)
```

## Troubleshooting

### Tests Failing

1. Check service dependencies (PostgreSQL, Redis)
2. Verify environment variables
3. Check test logs in Actions output

### Docker Build Failing

1. Check Dockerfile syntax
2. Verify build context
3. Check for missing dependencies

### Deployment Failing

1. Verify Kubernetes credentials (KUBECONFIG secrets)
2. Check cluster connectivity
3. Verify namespace exists
4. Check resource limits

### Integration Tests Failing

1. Ensure services are healthy before tests
2. Check database migrations ran
3. Verify Redis connectivity
4. Check service logs

## Customization

### Adding New Jobs

1. Create a new workflow file in `.github/workflows/`
2. Define triggers and jobs
3. Add to repository

### Modifying Deployment

1. Edit `cd.yml`
2. Update image tags
3. Adjust rollout strategies
4. Add smoke tests

### Adding Environments

1. Create environment in GitHub settings
2. Add to workflow file
3. Configure protection rules

## Best Practices

1. **Always test locally** before pushing
2. **Use feature branches** for development
3. **Review PRs** before merging
4. **Tag releases** for production deployments
5. **Monitor deployments** after release
6. **Rollback plan** for failed deployments
7. **Use semantic versioning** for tags
8. **Keep secrets secure** (never commit)
9. **Cache dependencies** for faster builds
10. **Run tests in parallel** when possible

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Buildx](https://docs.docker.com/buildx/)
- [Kubernetes Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)

