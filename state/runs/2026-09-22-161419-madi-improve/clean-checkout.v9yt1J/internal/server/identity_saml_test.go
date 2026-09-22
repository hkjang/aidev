package server

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
)

func samlTestResponse(t *testing.T, pair tls.Certificate, issuer, audience, acs, request, assertionID string, mutate func(*etree.Element)) []byte {
	t.Helper()
	escape := func(value string) string {
		var b bytes.Buffer
		_ = xml.EscapeText(&b, []byte(value))
		return b.String()
	}
	now := time.Now().UTC()
	document := etree.NewDocument()
	raw := fmt.Sprintf(`<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_%s" Version="2.0" IssueInstant="%s" Destination="%s" InResponseTo="%s"><saml:Issuer>%s</saml:Issuer><samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></samlp:Status><saml:Assertion ID="%s" Version="2.0" IssueInstant="%s"><saml:Issuer>%s</saml:Issuer><saml:Subject><saml:NameID Format="urn:oasis:names:tc:SAML:2.0:nameid-format:persistent">saml-worker</saml:NameID><saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer"><saml:SubjectConfirmationData InResponseTo="%s" Recipient="%s" NotOnOrAfter="%s"/></saml:SubjectConfirmation></saml:Subject><saml:Conditions NotBefore="%s" NotOnOrAfter="%s"><saml:AudienceRestriction><saml:Audience>%s</saml:Audience></saml:AudienceRestriction></saml:Conditions><saml:AuthnStatement AuthnInstant="%s"><saml:AuthnContext><saml:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport</saml:AuthnContextClassRef></saml:AuthnContext></saml:AuthnStatement><saml:AttributeStatement><saml:Attribute Name="email"><saml:AttributeValue>saml@example.test</saml:AttributeValue></saml:Attribute><saml:Attribute Name="displayName"><saml:AttributeValue>연합 사용자</saml:AttributeValue></saml:Attribute><saml:Attribute Name="groups"><saml:AttributeValue>Editors</saml:AttributeValue></saml:Attribute></saml:AttributeStatement></saml:Assertion></samlp:Response>`, newID(), now.Format(time.RFC3339), escape(acs), escape(request), escape(issuer), escape(assertionID), now.Format(time.RFC3339), escape(issuer), escape(request), escape(acs), now.Add(3*time.Minute).Format(time.RFC3339), now.Add(-time.Minute).Format(time.RFC3339), now.Add(3*time.Minute).Format(time.RFC3339), escape(audience), now.Format(time.RFC3339))
	if e := document.ReadFromString(raw); e != nil {
		t.Fatal(e)
	}
	assertion := document.Root().FindElement("./Assertion")
	assertion.CreateAttr("xmlns:saml", "urn:oasis:names:tc:SAML:2.0:assertion")
	if mutate != nil {
		mutate(assertion)
	}
	signing, e := dsig.NewSigningContext(pair.PrivateKey.(*rsa.PrivateKey), pair.Certificate)
	if e != nil {
		t.Fatal(e)
	}
	signing.IdAttribute = "ID"
	signing.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
	if e = signing.SetSignatureMethod(dsig.RSASHA256SignatureMethod); e != nil {
		t.Fatal(e)
	}
	signed, e := signing.SignEnveloped(assertion)
	if e != nil {
		t.Fatal(e)
	}
	document.Root().RemoveChild(assertion)
	document.Root().AddChild(signed)
	result, e := document.WriteToBytes()
	if e != nil {
		t.Fatal(e)
	}
	return result
}

func TestPostgresSAMLSignaturesStateAudienceAndRotation(t *testing.T) {
	s, admin, _, wid, _ := collaborationTestSetup(t)
	server := httptest.NewTLSServer(s)
	t.Cleanup(server.Close)
	browser := newIntegrationTestClient(t, server.URL)
	browser.client.Transport = server.Client().Transport
	browser.client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"site_url": server.URL}, 200)
	generated := testJSONObject(t, admin.request("POST", "/api/v1/admin/identity/saml/certificate", map[string]any{}, 200))
	pair, _ := identityTestTLS(t)
	issuer := "https://idp.example.test/entity"
	metadata := fmt.Sprintf(`<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s"><IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"><KeyDescriptor use="signing"><KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#"><X509Data><X509Certificate>%s</X509Certificate></X509Data></KeyInfo></KeyDescriptor><SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.test/sso"/></IDPSSODescriptor></EntityDescriptor>`, issuer, base64.StdEncoding.EncodeToString(pair.Certificate[0]))
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"saml_enabled": true, "saml_idp_metadata": metadata, "saml_auto_register": true, "identity_group_mappings": []any{map[string]any{"provider": "saml", "group": "Editors", "workspace_id": wid, "role": "editor"}}}, 200)
	start := func() (string, string) {
		t.Helper()
		response, e := browser.client.Get(server.URL + "/api/v1/auth/saml/start")
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		if response.StatusCode != 302 {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("start: %d %s", response.StatusCode, body)
		}
		redirect, e := url.Parse(response.Header.Get("Location"))
		if e != nil {
			t.Fatal(e)
		}
		query := redirect.Query()
		cfg, _ := s.settings(context.Background())
		sp, e := samlConfiguration(cfg, true)
		if e != nil {
			t.Fatal(e)
		}
		payload := "SAMLRequest=" + url.QueryEscape(query.Get("SAMLRequest")) + "&RelayState=" + url.QueryEscape(query.Get("RelayState")) + "&SigAlg=" + url.QueryEscape(query.Get("SigAlg"))
		hash := sha256.Sum256([]byte(payload))
		signature, _ := base64.StdEncoding.DecodeString(query.Get("Signature"))
		if e = rsa.VerifyPKCS1v15(sp.Certificate.PublicKey.(*rsa.PublicKey), crypto.SHA256, hash[:], signature); e != nil {
			t.Fatalf("AuthnRequest signature: %v", e)
		}
		encoded, _ := base64.StdEncoding.DecodeString(query.Get("SAMLRequest"))
		reader := flate.NewReader(bytes.NewReader(encoded))
		data, _ := io.ReadAll(reader)
		reader.Close()
		var request saml.AuthnRequest
		if e = xml.Unmarshal(data, &request); e != nil {
			t.Fatal(e)
		}
		return query.Get("RelayState"), request.ID
	}
	post := func(state string, data []byte, want int) {
		t.Helper()
		response, e := browser.client.PostForm(server.URL+"/api/v1/auth/saml/acs", url.Values{"RelayState": {state}, "SAMLResponse": {base64.StdEncoding.EncodeToString(data)}})
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		if response.StatusCode != want {
			t.Fatalf("ACS: %d want %d: %s", response.StatusCode, want, body)
		}
	}
	audience, acs := server.URL+"/api/v1/auth/saml/metadata", server.URL+"/api/v1/auth/saml/acs"
	state, request := start()
	assertionID := "_" + newID()
	good := samlTestResponse(t, pair, issuer, audience, acs, request, assertionID, nil)
	cfg, _ := s.settings(context.Background())
	sp, err := samlConfiguration(cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parseSAMLAssertion(sp, good, request); err != nil {
		if private, ok := err.(*saml.InvalidResponseError); ok {
			t.Fatalf("synthetic SAML fixture: %v", private.PrivateErr)
		}
		t.Fatal(err)
	}
	attacker := server.Client()
	attacker.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	stolen, e := attacker.PostForm(server.URL+"/api/v1/auth/saml/acs", url.Values{"RelayState": {state}, "SAMLResponse": {base64.StdEncoding.EncodeToString(good)}})
	if e != nil {
		t.Fatal(e)
	}
	stolen.Body.Close()
	if stolen.StatusCode != 400 {
		t.Fatal("SAML state not browser-bound")
	}
	post(state, good, 302)
	me := testJSONObject(t, browser.request("GET", "/api/v1/auth/me", nil, 200))
	if str(me, "email") != "saml@example.test" {
		t.Fatalf("wrong user: %v", me)
	}
	post(state, good, 400)
	state, request = start()
	post(state, samlTestResponse(t, pair, issuer, audience, acs, request, assertionID, nil), 401)
	for _, tc := range []struct {
		name   string
		mutate func(*etree.Element)
	}{
		{"audience", func(a *etree.Element) {
			a.FindElement("./Conditions/AudienceRestriction/Audience").SetText("https://other.test")
		}},
		{"missing audience", func(a *etree.Element) {
			c := a.FindElement("./Conditions")
			c.RemoveChild(c.FindElement("./AudienceRestriction"))
		}},
		{"expired", func(a *etree.Element) {
			a.FindElement("./Conditions").CreateAttr("NotOnOrAfter", time.Now().Add(-10*time.Minute).UTC().Format(time.RFC3339))
		}},
		{"subject missing", func(a *etree.Element) { a.RemoveChild(a.FindElement("./Subject")) }},
		{"confirmation mismatch", func(a *etree.Element) {
			a.FindElement("./Subject/SubjectConfirmation/SubjectConfirmationData").CreateAttr("InResponseTo", "_wrong")
		}},
		{"future assertion", func(a *etree.Element) {
			a.CreateAttr("IssueInstant", time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, request = start()
			post(state, samlTestResponse(t, pair, issuer, audience, acs, request, "_"+newID(), tc.mutate), 401)
		})
	}
	state, request = start()
	untrusted, _ := identityTestTLS(t)
	post(state, samlTestResponse(t, untrusted, issuer, audience, acs, request, "_"+newID(), nil), 401)
	state, request = start()
	post(state, samlTestResponse(t, pair, issuer, audience, acs, request, "_"+newID(), func(a *etree.Element) {
		a.FindElement("./Subject/NameID").SetText("different-unbound-admin-subject")
		a.FindElement("./AttributeStatement/Attribute[@Name='email']/AttributeValue").SetText("admin@example.test")
	}), 403)
	state, request = start()
	admin.request("POST", "/api/v1/admin/identity/saml/certificate", map[string]any{}, 409)
	rotated := testJSONObject(t, admin.request("POST", "/api/v1/admin/identity/saml/certificate", map[string]any{"confirm": true}, 200))
	if str(rotated, "certificate") == str(generated, "certificate") {
		t.Fatal("certificate was not rotated")
	}
	post(state, samlTestResponse(t, pair, issuer, audience, acs, request, "_"+newID(), nil), 409)
	var stored string
	if e := s.DB.QueryRow(context.Background(), `SELECT data::text FROM settings WHERE id=1`).Scan(&stored); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(stored, "BEGIN PRIVATE KEY") {
		t.Fatal("unencrypted SAML signing key")
	}
}
