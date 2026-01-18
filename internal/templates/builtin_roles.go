package templates

// BuiltinRole represents a built-in role template
type BuiltinRole struct {
	Name        string
	Description string
	Content     string
}

// GetBuiltinRoles returns all built-in role templates
func GetBuiltinRoles() []BuiltinRole {
	return []BuiltinRole{
		{
			Name:        "frontend",
			Description: "Frontend development (Vue/React/TypeScript)",
			Content:     frontendRole,
		},
		{
			Name:        "backend",
			Description: "Backend development (Python/FastAPI/Go)",
			Content:     backendRole,
		},
		{
			Name:        "test",
			Description: "Testing and E2E (Vitest/Playwright/pytest)",
			Content:     testRole,
		},
		{
			Name:        "docs",
			Description: "Documentation (Markdown/README)",
			Content:     docsRole,
		},
		{
			Name:        "security",
			Description: "Security review and implementation",
			Content:     securityRole,
		},
		{
			Name:        "devops",
			Description: "CI/CD and infrastructure",
			Content:     devopsRole,
		},
	}
}

// GetBuiltinRole returns a specific built-in role by name
func GetBuiltinRole(name string) *BuiltinRole {
	for _, role := range GetBuiltinRoles() {
		if role.Name == name {
			return &role
		}
	}
	return nil
}

// IsBuiltinRole checks if a role name is a built-in role
func IsBuiltinRole(name string) bool {
	return GetBuiltinRole(name) != nil
}

const frontendRole = `# Frontend Worker

## Basic Rules
- Think and output in the project's primary language
- Only modify files under frontend/ directory
- Follow existing code patterns and style

## Technology Stack
- Framework: Vue 3 (Composition API) / React
- Language: TypeScript
- State Management: Pinia / Redux / Context
- UI Library: Vuetify / Material UI / Tailwind

## Commands
- Development: npm run dev
- Test: npm run test / npm run test:unit
- Lint: npm run lint
- Build: npm run build

## Best Practices
- Use TypeScript strictly (no any)
- Write unit tests for components
- Follow accessibility guidelines (a11y)
- Optimize bundle size
`

const backendRole = `# Backend Worker

## Basic Rules
- Think and output in the project's primary language
- Only modify files under backend/ or api/ directory
- Follow existing code patterns and style

## Technology Stack
- Language: Python / Go / Node.js
- Framework: FastAPI / Gin / Express
- Database: PostgreSQL / SQLite
- ORM: SQLAlchemy / GORM

## Commands
- Development: python -m uvicorn main:app --reload / go run .
- Test: pytest / go test ./...
- Lint: ruff check / golangci-lint run

## Best Practices
- Write comprehensive API documentation
- Add proper error handling
- Write unit and integration tests
- Use proper logging
- Follow security best practices
`

const testRole = `# Test Worker

## Basic Rules
- Think and output in the project's primary language
- Focus on test coverage and quality
- Maintain test documentation

## Technology Stack
- Unit Testing: Vitest / Jest / pytest
- E2E Testing: Playwright / Cypress
- API Testing: pytest / supertest

## Commands
- Unit tests: npm run test:unit / pytest
- E2E tests: npm run test:e2e / playwright test
- Coverage: npm run test:coverage / pytest --cov

## Best Practices
- Aim for high test coverage (>80%)
- Write meaningful test descriptions
- Use proper test isolation
- Mock external dependencies
- Test edge cases and error conditions
`

const docsRole = `# Documentation Worker

## Basic Rules
- Think and output in the project's primary language
- Keep documentation clear and concise
- Update docs when code changes

## Areas of Responsibility
- README files
- API documentation
- User guides
- Architecture documentation
- Code comments

## Format Guidelines
- Use Markdown format
- Include code examples
- Add diagrams when helpful
- Keep language simple and clear

## Best Practices
- Keep docs in sync with code
- Use consistent terminology
- Include version information
- Provide examples for complex features
`

const securityRole = `# Security Worker

## Basic Rules
- Think and output in the project's primary language
- Prioritize security in all decisions
- Document security considerations

## Areas of Responsibility
- Authentication and Authorization
- Input validation
- CORS configuration
- Secrets management
- Dependency security

## Security Checklist
- [ ] No hardcoded secrets
- [ ] Input validation on all endpoints
- [ ] Proper authentication checks
- [ ] Rate limiting implemented
- [ ] SQL injection prevention
- [ ] XSS prevention
- [ ] CSRF protection

## Best Practices
- Follow OWASP guidelines
- Regular dependency updates
- Security-focused code review
- Proper logging (no sensitive data)
`

const devopsRole = `# DevOps Worker

## Basic Rules
- Think and output in the project's primary language
- Focus on reliability and automation
- Document infrastructure changes

## Areas of Responsibility
- CI/CD pipelines
- Docker configuration
- Infrastructure as Code
- Monitoring and logging
- Deployment automation

## Technology Stack
- CI/CD: GitHub Actions / GitLab CI
- Containers: Docker / Docker Compose
- IaC: Terraform / Pulumi
- Monitoring: Prometheus / Grafana

## Best Practices
- Infrastructure as Code
- Automated testing in CI
- Blue-green deployments
- Proper secret management
- Monitoring and alerting
`
