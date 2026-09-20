package main

import (
"fmt"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"regexp"
	"strings"
)

// PGP writes the only header that does not end at "PRIVATE KEY": an exported
// secret key is "-----BEGIN PGP PRIVATE KEY BLOCK-----". A .asc or .gpg file
// committed to a repository was therefore indexed whole, key material included,
// while every other private key format was blocked.
var privateKeyRE = regexp.MustCompile(`(?i)-----BEGIN(?: [A-Z0-9]+)* PRIVATE KEY(?: BLOCK)?-----`)

// Naming the field is the only signal available for values the entropy rule
// cannot reach -- notably hex, whose entropy tops out at 4.0 and so never
// clears the 4.2 gate. The bare "token" and "secret" alternatives catch names
// like MY_TOKEN, and trail the compound forms because a bare "secret" cannot
// span the underscore in SECRET_KEY.
//
// The name may be quoted. JSON, Terraform and every config file written in them
// spell the pair as "password": "hunter2", and reading only the bare form put
// the closing quote between the name and the colon, so the rule never matched:
// an appsettings.json or a Postman collection was indexed, and returned in a
// snippet, with its credential intact. The quotes are optional on both sides,
// so the bare form still matches exactly as it did.
//
// An assignment is one line. Written with \s the rule ran past the end of it
// and took the next line's first word as the value, which YAML makes constant:
// "secret:" and "token:" are ordinary section headers, and a Kubernetes volume
// that says
//
//	secret:
//	  secretName: db-creds
//
// came back as "secret: [REDACTED] db-creds" — the key masked, the value left,
// a security event raised for a manifest holding no credential, and a line gone
// from the chunk. The line matters beyond the text: chunks are cut from the
// masked content, so every line number after a swallowed newline described the
// wrong line of the real file for the rest of that file.
var secretAssignmentRE = regexp.MustCompile(`(?i)["']?(` + secretNames + `)["']?[^\S\r\n]*[:=][^\S\r\n]*(?:"[^"\r\n]{4,}"|'[^'\r\n]{4,}'|[^\s,;#]{4,})`)

// secretNames is the list of field names shared by the rules that recognise a
// credential by what it is called. Keeping one list means a name added for one
// syntax is understood in the others too.
const secretNames = `api[_-]?key|secret[_-]?key|client[_-]?secret|access[_-]?token|auth[_-]?token|refresh[_-]?token|private[_-]?token|password|passwd|passphrase|credential|token|secret`

// A YAML block scalar states the value on the lines after the key, and no
// assignment rule can reach it: "password: |" has one character where the rule
// wants four, and the lines that follow say nothing about what they belong to.
// The shape is how a manifest writes a credential that does not fit on one line
// or contains characters it would rather not quote — Kubernetes and Helm values,
// Ansible vars, GitLab CI variables, docker-compose configs — and the body went
// into the index and back out in a snippet in full. A key and a certificate are
// caught by the private key rule, and a long random token by the entropy rule,
// so what was left readable is exactly the plain password and the short token.
//
// The indicator ends the line: "|", ">", a chomping "-" or "+", an indentation
// digit, and nothing else but a comment. Requiring that keeps the rule off the
// "token: > 5" of a template language and off a Markdown table, whose cells
// begin with the pipe rather than end with it.
var blockScalarSecretRE = regexp.MustCompile(`(?im)^([^\S\r\n]*)(?:-[^\S\r\n]+)?["']?(?:` + secretNames + `)["']?[^\S\r\n]*:[^\S\r\n]*[|>][-+0-9]*[^\S\r\n]*(?:#[^\r\n]*)?\r?$`)
var awsKeyRE = regexp.MustCompile(`\bAKIA[A-Z0-9]{16}\b`)

// Vendor prefixes are matched explicitly rather than left to the entropy rule,
// which mislabels them and misses the segments shorter than 32 characters.
var knownTokenRE = regexp.MustCompile(`\b(?:glpat-[A-Za-z0-9_-]{20,}|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{20,}|hvs\.[A-Za-z0-9_-]{20,}|xox[baprs]-[A-Za-z0-9-]{10,}|[sr]k_(?:live|test)_[A-Za-z0-9]{16,}|sk-(?:proj-)?[A-Za-z0-9_-]{20,}|AIza[A-Za-z0-9_-]{35}|SG\.[A-Za-z0-9_-]{16,}\.[A-Za-z0-9_-]{16,}|npm_[A-Za-z0-9]{36}|eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,})\b`)

// A credential in a URL is not a database problem, and listing schemes made it
// look like one. Three were listed -- postgres, mysql, mariadb -- so
// mongodb://, redis://, amqp://, sqlserver:// and, above all,
// https://user:token@host went through untouched. A clone URL carrying an
// access token is the single most common way a credential appears in the
// repositories this platform indexes: it sits in READMEs, CI files and setup
// notes. Any scheme counts now.
//
// Only the userinfo is replaced. The host is what makes the finding
// actionable -- "a token for bitbucket.company is in this file" -- and hiding
// it would take the useful half of the line with the dangerous half.
// The password half deliberately admits "@": P@ssw0rd is a password people
// actually choose, and stopping at the first "@" left "ssw0rd" in the answer.
// It still cannot cross a space or a slash, so it stays inside one URL.
//
// The user half may be empty. Redis, AMQP and the clients built on them
// authenticate with a password and no user name, so their URLs are written
// redis://:s3cr3t@cache:6379 — requiring at least one character before the
// colon meant the one credential in the line was the one part not matched.
var credentialURLRE = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^\s/:@"'<>]*:[^\s/"'<>]+@`)

// Oracle's thin driver puts the credentials before the @ with a slash rather
// than a colon, so no URL rule reaches it. It is too common in the
// installations this platform is built for to leave to the entropy rule, which
// never sees a password like "tiger" at all.
var oracleDSNRE = regexp.MustCompile(`(?i)\b(jdbc:oracle:[a-z]+:)[^\s/@"']+/[^\s@"']+@`)

// Java and Spring configuration states a credential as an element or a pair of
// attributes, neither of which is an assignment. <password>value</password> and
// <property name="password" value="value"/> were both returned in full.
//
// An element usually carries a namespace prefix in the files this platform
// indexes -- soapUI projects, WSDL bindings, WebSphere and Spring descriptors
// all write <con:password> -- and requiring the name to follow "<" directly
// missed every one of them. The prefix is captured with the name so the closing
// tag still matches the opening one after the value is replaced.
var xmlSecretElementRE = regexp.MustCompile(`(?i)<((?:[a-z0-9_.-]+:)?(?:password|passwd|secret|token|api[_-]?key|client[_-]?secret|credential))\s*>([^<\r\n]{4,})</`)
var xmlSecretAttributeRE = regexp.MustCompile(`(?i)(name\s*=\s*"[^"\r\n]*(?:password|passwd|secret|token|credential)[^"\r\n]*"\s+value\s*=\s*")[^"\r\n]{4,}"`)

// A command line is where a credential is most often written down for someone
// else to copy, and curl's -u takes it as user:password. The curl anchor keeps
// this away from every other tool that happens to have a -u flag.
var curlUserRE = regexp.MustCompile(`(?i)\bcurl\b[^\n]{0,200}?(\s-u\s+|\s--user\s+)[^\s:"']+:[^\s"']+`)

// .netrc and its imitators separate the value with spaces rather than a colon
// or an equals sign, which no assignment rule reaches. Requiring the login
// field before it keeps the rule off prose that merely says "password".
//
// A real .netrc puts each field on its own line, so the separators are captured
// and written back rather than replaced by single spaces: rewriting the whole
// match collapsed the three-line entry onto one and moved every line after it.
var netrcRE = regexp.MustCompile(`(?i)\b(login\s+)\S+(\s+password\s+)\S{4,}`)

// An Authorization header carries a credential whose shape is the issuer's
// business, so no vendor prefix and no entropy floor will find it.
var authorizationHeaderRE = regexp.MustCompile(`(?i)\b(authorization\s*[:=]\s*)(bearer|basic|token)\s+[^\s"'<>]{8,}`)
var entropyCandidateRE = regexp.MustCompile(`[A-Za-z0-9+/=_-]{32,}`)
var commonHashRE = regexp.MustCompile(`(?i)^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)

// Revision fingerprints the masking rules themselves.
//
// A stored chunk was masked by the rules in force when it was indexed, so
// improving the rules does nothing for content already in the database — the
// credentials a new rule catches stay readable until that ref is read again.
// The indexer already re-reads a ref whose policy fingerprint moved; this makes
// a rule change the same kind of event.
//
// It is derived from the patterns rather than written down, because a constant
// somebody has to remember to bump is a constant that will be forgotten, and
// the failure is silent.
func Revision() string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		privateKeyRE.String(), secretAssignmentRE.String(), awsKeyRE.String(), knownTokenRE.String(),
		credentialURLRE.String(), oracleDSNRE.String(), xmlSecretElementRE.String(), xmlSecretAttributeRE.String(),
		curlUserRE.String(), netrcRE.String(), authorizationHeaderRE.String(), entropyCandidateRE.String(),
		commonHashRE.String(), blockScalarSecretRE.String(),
	}, "\x00")))
	return hex.EncodeToString(sum[:6])
}

// Sanitize masks secrets in untrusted repository content before indexing or
// returning a snippet received directly from a remote source search API.
func Sanitize(content string) (string, string) {
	if privateKeyRE.MatchString(content) {
		return "", "private_key"
	}
	finding := ""
	masked, hidden := maskBlockScalars(content)
	if hidden {
		finding = "credential_assignment"
	}
	masked = secretAssignmentRE.ReplaceAllStringFunc(masked, func(value string) string {
		finding = "credential_assignment"
		at := strings.IndexAny(value, ":=")
		if at < 0 {
			return "[REDACTED]"
		}
		return value[:at+1] + " [REDACTED]"
	})
	masked = awsKeyRE.ReplaceAllStringFunc(masked, func(string) string { finding = "cloud_access_key"; return "[REDACTED]" })
	masked = knownTokenRE.ReplaceAllStringFunc(masked, func(string) string { finding = "known_token"; return "[REDACTED]" })
	masked = credentialURLRE.ReplaceAllStringFunc(masked, func(value string) string {
		finding = "credential_dsn"
		return credentialURLRE.ReplaceAllString(value, "${1}[REDACTED]@")
	})
	masked = oracleDSNRE.ReplaceAllStringFunc(masked, func(value string) string {
		finding = "credential_dsn"
		return oracleDSNRE.ReplaceAllString(value, "${1}[REDACTED]@")
	})
	masked = xmlSecretElementRE.ReplaceAllStringFunc(masked, func(value string) string {
		finding = "credential_assignment"
		return xmlSecretElementRE.ReplaceAllString(value, "<${1}>[REDACTED]</")
	})
	masked = xmlSecretAttributeRE.ReplaceAllStringFunc(masked, func(value string) string {
		finding = "credential_assignment"
		return xmlSecretAttributeRE.ReplaceAllString(value, `${1}[REDACTED]"`)
	})
	masked = curlUserRE.ReplaceAllStringFunc(masked, func(value string) string {
		finding = "credential_assignment"
		return curlUserRE.ReplaceAllString(value, "${1}[REDACTED]")
	})
	masked = netrcRE.ReplaceAllStringFunc(masked, func(value string) string {
		finding = "credential_assignment"
		return netrcRE.ReplaceAllString(value, "${1}[REDACTED]${2}[REDACTED]")
	})
	masked = authorizationHeaderRE.ReplaceAllStringFunc(masked, func(value string) string {
		finding = "credential_assignment"
		return authorizationHeaderRE.ReplaceAllString(value, "${1}${2} [REDACTED]")
	})
	masked = entropyCandidateRE.ReplaceAllStringFunc(masked, func(value string) string {
		if commonHashRE.MatchString(value) || shannonEntropy(value) < 4.2 {
			return value
		}
		finding = "high_entropy_secret"
		return "[REDACTED_HIGH_ENTROPY]"
	})
	return masked, finding
}

// maskBlockScalars replaces the body of every block scalar whose key names a
// credential, and reports whether it replaced anything.
//
// The body is the run of lines indented deeper than the key, which is how YAML
// itself decides where the value ends, so a dedent leaves the block and the rest
// of the document is untouched. Each line is replaced on its own, keeping its
// indentation: chunks are cut from the masked content and stored with the line
// numbers they came from, so joining the block into one [REDACTED] would move
// every line after it in that file.
func maskBlockScalars(content string) (string, bool) {
	if !blockScalarSecretRE.MatchString(content) {
		return content, false
	}
	lines := strings.Split(content, "\n")
	masked := false
	for index := 0; index < len(lines); index++ {
		header, _ := splitReturn(lines[index])
		if !blockScalarSecretRE.MatchString(header) {
			continue
		}
		depth := blockIndent(header)
		end := index + 1
		for ; end < len(lines); end++ {
			body, carriage := splitReturn(lines[end])
			// A blank line inside a block is part of the block, and blanking it
			// again would say nothing.
			if strings.TrimSpace(body) == "" {
				continue
			}
			indent := blockIndent(body)
			if indent <= depth {
				break
			}
			lines[end] = body[:indent] + "[REDACTED]" + carriage
			masked = true
		}
		index = end - 1
	}
	if !masked {
		return content, false
	}
	return strings.Join(lines, "\n"), true
}

// splitReturn separates a line from the carriage return of a CRLF file, so a
// replacement can be written back without changing the line ending.
func splitReturn(line string) (string, string) {
	if strings.HasSuffix(line, "\r") {
		return line[:len(line)-1], "\r"
	}
	return line, ""
}

func blockIndent(line string) int { return len(line) - len(strings.TrimLeft(line, " \t")) }

func shannonEntropy(value string) float64 {
	if value == "" {
		return 0
	}
	counts := map[rune]int{}
	for _, char := range value {
		counts[char]++
	}
	length := float64(len([]rune(value)))
	var entropy float64
	for _, count := range counts {
		probability := float64(count) / length
		entropy -= probability * math.Log2(probability)
	}
	return entropy
}

func main() { for _, input := range []string{`curl -u admin:Passw0rd https://api.company/v1/health`, `curl --silent --user admin:Passw0rd https://api.company/v1/health`, "- password: |\n    demo-value\n  name: public-label\n"} { out, finding := Sanitize(input); fmt.Printf("input=%q\noutput=%q finding=%q\n", input,out,finding) }; fmt.Println("revision="+Revision()) }
