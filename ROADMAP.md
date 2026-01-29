# OAS Roadmap

## Authetication Support
- [ ] API keys (header, query, cookie)
- [ ] Bearer tokens / JWT
- [ ] Basic auth
- [ ] OAuth2 flows
- [ ] Custom header injection via --header "Authorization: Bearer xxx"

## Environment Variables & Profiles
- [ ] Support .env files and environment variable substitution
- [ ] Named profiles (--profile staging, --profile prod)
- [ ] Variable interpolation in server URLs and headers

## Chained Requests / Workflows
- [ ] Extract values from responses to use in subsequent requests
- [ ] Define test sequences (e.g., create → read → update → delete)
- [ ] Dependency resolution between operations

## HTML Report Generation
- [ ] --output html for rich visual reports
- [ ] Charts for benchmark latency distribution
- [ ] Shareable test reports

## Parallel Endpoint Testing
- [ ] Test multiple endpoints simultaneously (not just concurrent requests to one endpoint)
- [ ] --parallel-endpoints flag

## Spec Validation Command
- [ ] oas validate api-spec.json - validate OpenAPI spec before testing
- [ ] Report spec issues and warnings
