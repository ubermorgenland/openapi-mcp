# Pull Request

## 📝 Description

<!-- Provide a clear and concise description of what this PR accomplishes -->

## 🎯 Type of Change

<!-- Mark the relevant option with an "x" -->

- [ ] 🐛 Bug fix (non-breaking change that fixes an issue)
- [ ] ✨ New feature (non-breaking change that adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📚 Documentation update (improvements to docs, examples, or guides)
- [ ] 🧪 Test improvement (adding or improving tests)
- [ ] 🔧 Refactoring (code change that neither fixes a bug nor adds a feature)
- [ ] ⚡ Performance improvement (non-breaking change that improves performance)
- [ ] 🏗️ Build/CI improvement (changes to build process or CI/CD)

## 🔗 Related Issues

<!-- Link any related issues using "Fixes #123", "Closes #456", "Related to #789" -->

- Fixes #
- Related to #

## 🧪 Testing

### Test Coverage
<!-- Describe how your changes are tested -->

- [ ] All existing tests pass (`go test ./...`)
- [ ] New tests added for new functionality
- [ ] Manual testing completed
- [ ] Integration tests updated (if applicable)

### Manual Testing Performed
<!-- Describe manual testing you've done -->

```sh
# Example commands you used to test
bin/openapi-mcp examples/weather.yaml
curl -X POST http://localhost:8080/mcp -d '...'
```

**Test Results:**
<!-- Describe what you tested and the results -->

## 📋 Changes Made

### Code Changes
<!-- List the main code changes -->

- [ ] Modified: `pkg/openapi2mcp/...` - Description of changes
- [ ] Added: `pkg/newpackage/...` - Description of new functionality  
- [ ] Fixed: `cmd/openapi-mcp/...` - Description of bug fix
- [ ] Removed: `deprecated_file.go` - Reason for removal

### Documentation Changes
<!-- List documentation updates -->

- [ ] Updated README.md with new feature documentation
- [ ] Added godoc comments to new exported functions
- [ ] Created/updated examples in `examples/`
- [ ] Updated `CONTRIBUTING.md` or other guides

### Database Changes (if applicable)
<!-- If your changes affect database functionality -->

- [ ] Database schema changes
- [ ] Migration scripts provided
- [ ] Seed data updated
- [ ] Database documentation updated

## ⚠️ Breaking Changes

<!-- If this PR includes breaking changes, describe them here -->

**Breaking Changes:**
- None

**Migration Guide** (if applicable):
<!-- Provide steps users need to take to migrate -->

## 🔍 Checklist

### Code Quality
- [ ] Code follows existing style and conventions
- [ ] All functions have appropriate godoc comments
- [ ] Error handling is comprehensive and clear
- [ ] No hardcoded secrets or sensitive data
- [ ] Code is properly formatted (`go fmt ./...`)
- [ ] No linting errors (`go vet ./...`)

### Testing
- [ ] All tests pass locally
- [ ] New functionality has test coverage
- [ ] Edge cases are tested
- [ ] Integration tests pass (if applicable)

### Documentation
- [ ] README updated for new features
- [ ] Godoc comments added for new exported items
- [ ] Examples updated or added
- [ ] Breaking changes documented

### Security
- [ ] No sensitive data exposed in code or tests
- [ ] Authentication changes are secure
- [ ] Input validation is present where needed
- [ ] Security implications considered and documented

## 📸 Screenshots (if applicable)

<!-- Include screenshots for UI changes, output examples, etc. -->

## 📈 Performance Impact

<!-- Describe any performance implications -->

- [ ] No performance impact
- [ ] Minor performance improvement
- [ ] Minor performance regression (justified by benefits)
- [ ] Significant performance change (benchmarks provided below)

**Benchmarks** (if applicable):
```
<!-- Paste benchmark results -->
```

## 🎉 Additional Notes

<!-- Any additional information that would help reviewers -->

### For Reviewers
<!-- Anything specific you'd like reviewers to focus on -->

### Future Work
<!-- Any follow-up work or related improvements planned -->

---

**Thank you for contributing to openapi-mcp!** 🙏

<!-- 
Reviewer Guidelines:
- Focus on code correctness, security, and maintainability
- Check that tests are comprehensive and meaningful
- Verify documentation is clear and complete
- Consider backwards compatibility and API stability
- Ensure changes align with project goals and architecture
-->