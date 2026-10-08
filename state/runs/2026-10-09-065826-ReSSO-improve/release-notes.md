## What's Changed
* Reject a missing or unreadable CA when reusing the TLS test directory, and document recovery using the existing certificate mount, by @hkjang in https://github.com/hkjang/ReSSO/pull/41
* Preserve preparation failures in the LDAP development guide with assignment, evaluation and tests joined by `&&`; clarify service reuse, optional cleanup and the shared certificate path, by @hkjang in https://github.com/hkjang/ReSSO/pull/42

## Upgrade notes
This release updates development test preparation and its documentation. There are no production Go code, database schema or runtime configuration changes. Existing installations can roll back to v0.9.102.

Developers reusing an LDAPS container must point `RESSO_TEST_CERT_DIR` at its existing readable CA directory. Preparation and cleanup must use the same path. Cleanup removes test data and remains optional.

The tag-triggered workflow builds and verifies the linux/amd64 offline image, then attaches `resso-v0.9.103.tar.gz` and `release-sha256.txt` and supplies the archive SHA-256. Artifact publication is pending that workflow.

## Validation
Local release gates passed: `make lint`, `make test`, `make build VERSION=v0.9.103`, release version consistency and `git diff --check`. Go race tests passed in 13 packages (some cached), and frontend tests passed in 29 files with 161 tests. The fresh TLS certificate verification test passed without SKIP. The guide's missing-CA probe returned nonzero without running its follow-up. Test containers and CA metadata were preserved.

npm still reports 5 vulnerabilities (2 moderate, 3 high); govulncheck reports 3 module vulnerabilities not called by this code. These were not remediated in this release. No total zero-SKIP claim is made for the nonverbose full Go suite.

**Full Changelog**: https://github.com/hkjang/ReSSO/compare/v0.9.102...v0.9.103
