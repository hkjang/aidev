package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
)

func identityTestTLS(t *testing.T) (tls.Certificate, string) {
	t.Helper()
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, e := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	certificate := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, certificate
}

func identityTestLDAP(t *testing.T) (string, string, func(string)) {
	t.Helper()
	certificate, ca := identityTestTLS(t)
	listener, e := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { listener.Close() })
	var mu sync.Mutex
	email := "ldap@example.test"
	send := func(connection net.Conn, id int64, kind ber.Tag, code int64) {
		message := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "")
		message.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, id, ""))
		response := ber.Encode(ber.ClassApplication, ber.TypeConstructed, kind, nil, "")
		response.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagEnumerated, code, ""))
		response.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", ""))
		response.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", ""))
		message.AppendChild(response)
		connection.Write(message.Bytes())
	}
	go func() {
		for {
			connection, e := listener.Accept()
			if e != nil {
				return
			}
			go func() {
				defer connection.Close()
				for {
					packet, e := ber.ReadPacket(connection)
					if e != nil {
						return
					}
					id := packet.Children[0].Value.(int64)
					request := packet.Children[1]
					switch request.Tag {
					case 0:
						dn := request.Children[1].Value.(string)
						password := string(request.Children[2].Data.Bytes())
						ok := (dn == "cn=reader,dc=example,dc=test" && password == "reader-secret") || (dn == "uid=worker,dc=example,dc=test" && password == "directory-secret")
						code := int64(49)
						if ok {
							code = 0
						}
						send(connection, id, 1, code)
					case 3:
						mu.Lock()
						value := email
						mu.Unlock()
						message := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "")
						message.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, id, ""))
						entry := ber.Encode(ber.ClassApplication, ber.TypeConstructed, 4, nil, "")
						entry.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "uid=worker,dc=example,dc=test", ""))
						attributes := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "")
						for name, value := range map[string]string{"entryUUID": "directory-worker-id", "mail": value, "displayName": "디렉터리 사용자", "memberOf": "CN=Editors,DC=example,DC=test"} {
							attribute := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "")
							attribute.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, name, ""))
							values := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSet, nil, "")
							values.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, value, ""))
							attribute.AppendChild(values)
							attributes.AppendChild(attribute)
						}
						entry.AppendChild(attributes)
						message.AppendChild(entry)
						connection.Write(message.Bytes())
						send(connection, id, 5, 0)
					case 2:
						return
					default:
						return
					}
				}
			}()
		}
	}()
	return "ldaps://" + listener.Addr().String(), ca, func(value string) { mu.Lock(); email = value; mu.Unlock() }
}

func TestPostgresLDAPTLSProvisioningAndManagedRoles(t *testing.T) {
	s, admin, _, wid, _ := collaborationTestSetup(t)
	endpoint, ca, setEmail := identityTestLDAP(t)
	mapping := []any{map[string]any{"provider": "ldap", "group": "CN=Editors,DC=example,DC=test", "workspace_id": wid, "role": "editor"}}
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ldap_enabled": true, "ldap_url": endpoint, "ldap_ca_pem": ca, "ldap_bind_dn": "cn=reader,dc=example,dc=test", "ldap_bind_password": "reader-secret", "ldap_base_dn": "dc=example,dc=test", "ldap_auto_register": true, "identity_group_mappings": mapping}, 200)
	browser := newIntegrationTestClient(t, admin.base)
	browser.request("POST", "/api/v1/auth/ldap/login", map[string]any{"username": "worker", "password": "incorrect"}, 401)
	user := testJSONObject(t, browser.request("POST", "/api/v1/auth/ldap/login", map[string]any{"username": "worker", "password": "directory-secret"}, 200))
	uid := str(user, "id")
	if str(user, "email") != "ldap@example.test" {
		t.Fatal(user)
	}
	var role string
	ctx := context.Background()
	if e := s.DB.QueryRow(ctx, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid).Scan(&role); e != nil || role != "editor" {
		t.Fatalf("LDAP mapping %s %v", role, e)
	}
	mapping[0].(map[string]any)["role"] = "viewer"
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"identity_group_mappings": mapping}, 200)
	if e := s.DB.QueryRow(ctx, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid).Scan(&role); e != nil || role != "viewer" {
		t.Fatalf("mapping change did not apply: %s %v", role, e)
	}
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "ldap@example.test", "role": "editor"}, 200)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"identity_group_mappings": []any{}}, 200)
	browser.request("POST", "/api/v1/auth/ldap/login", map[string]any{"username": "worker", "password": "directory-secret"}, 200)
	if e := s.DB.QueryRow(ctx, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid).Scan(&role); e != nil || role != "editor" {
		t.Fatalf("manual grant was revoked: %s %v", role, e)
	}
	var stored []byte
	if e := s.DB.QueryRow(ctx, `SELECT data FROM settings WHERE id=1`).Scan(&stored); e != nil || strings.Contains(string(stored), "reader-secret") {
		t.Fatal("LDAP secret not encrypted", e)
	}
	// A changed directory email cannot bind an existing local administrator.
	setEmail("admin@example.test")
	if _, e := s.DB.Exec(ctx, `DELETE FROM identity_links WHERE provider='ldap'`); e != nil {
		t.Fatal(e)
	}
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"identity_link_policy": "verified_email_non_admin"}, 200)
	browser.request("POST", "/api/v1/auth/ldap/login", map[string]any{"username": "worker", "password": "directory-secret"}, 403)
}

func TestLDAPRejectsPlaintextAndUntrustedCA(t *testing.T) {
	endpoint, _, _ := identityTestLDAP(t)
	cfg := defaultSettings()
	cfg["ldap_enabled"] = true
	cfg["ldap_url"] = endpoint
	cfg["ldap_base_dn"] = "dc=example,dc=test"
	if _, e := authenticateLDAP(context.Background(), cfg, "worker", "directory-secret"); e == nil {
		t.Fatal("untrusted TLS certificate accepted")
	}
	cfg["ldap_url"] = "ldap://127.0.0.1:389"
	cfg["ldap_starttls"] = false
	if _, _, e := ldapConfiguration(cfg); e == nil {
		t.Fatal("plaintext LDAP bind allowed")
	}
}
