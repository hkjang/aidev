package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"
	goora "github.com/sijms/go-ora/v2"
)

type sqlSource struct {
	ID, WorkspaceID, SpaceID, OwnerID, ServiceID, Name, Kind string
	Config, Credentials                                      map[string]any
	Enabled                                                  bool
	Revision                                                 int
}
type sqlSourceDial struct{ host, port string }

func (d sqlSourceDial) HostName() string { return d.host }
func (d sqlSourceDial) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, e := net.SplitHostPort(address)
	if e != nil || !strings.EqualFold(host, d.host) || port != d.port {
		return nil, errors.New("SQL 연결이 허용되지 않은 대상 주소를 요청했습니다")
	}
	return webhookDial(ctx, network, address)
}

func (s *Server) openSQLSource(ctx context.Context, c sqlSource) (*sql.DB, error) {
	host := str(c.Config, "host")
	port := strconv.Itoa(number(c.Config, "port", 0))
	plain := boolean(c.Config, "allow_plaintext")
	scheme := "https"
	if plain {
		scheme = "http"
	}
	probe := connectorConfig{BaseURL: (&url.URL{Scheme: scheme, Host: net.JoinHostPort(host, port)}).String(), Config: map[string]any{"allow_http": plain, "insecure_tls": boolean(c.Config, "insecure_tls"), "ca_pem": str(c.Config, "ca_pem")}}
	client, e := s.connectorHTTP(ctx, probe)
	if e != nil {
		return nil, e
	}
	tlsConfig := client.Transport.(*http.Transport).TLSClientConfig.Clone()
	tlsConfig.ServerName = host
	dial := sqlSourceDial{host: host, port: port}
	database := str(c.Config, "database")
	user := str(c.Credentials, "username")
	password := str(c.Credentials, "password")
	timeout := time.Duration(number(c.Config, "timeout_seconds", 15)) * time.Second
	var db *sql.DB
	switch c.Kind {
	case "postgres":
		u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(host, port), Path: "/" + database, User: url.UserPassword(user, password)}
		q := url.Values{"sslmode": {"require"}}
		if plain {
			q.Set("sslmode", "disable")
		}
		u.RawQuery = q.Encode()
		cfg, e := pgx.ParseConfig(u.String())
		if e != nil {
			return nil, errors.New("PostgreSQL 접속 설정 형식을 확인하세요")
		}
		cfg.DialFunc = dial.DialContext
		cfg.LookupFunc = func(_ context.Context, requested string) ([]string, error) {
			if !strings.EqualFold(requested, host) {
				return nil, errors.New("허용되지 않은 PostgreSQL 호스트입니다")
			}
			return []string{host}, nil
		}
		cfg.ConnectTimeout = timeout
		cfg.Fallbacks = nil
		cfg.RuntimeParams["application_name"] = "madi-readonly-source"
		cfg.RuntimeParams["default_transaction_read_only"] = "on"
		if !plain {
			cfg.TLSConfig = tlsConfig
		}
		db = stdlib.OpenDB(*cfg)
	case "mysql", "mariadb":
		cfg := mysql.NewConfig()
		cfg.User = user
		cfg.Passwd = password
		cfg.Net = "tcp"
		cfg.Addr = net.JoinHostPort(host, port)
		cfg.DBName = database
		cfg.Timeout = timeout
		cfg.ReadTimeout = timeout
		cfg.WriteTimeout = timeout
		cfg.DialFunc = dial.DialContext
		cfg.ParseTime = true
		cfg.MultiStatements = false
		cfg.AllowAllFiles = false
		cfg.AllowNativePasswords = true
		if !plain {
			cfg.TLS = tlsConfig
		}
		connector, e := mysql.NewConnector(cfg)
		if e != nil {
			return nil, errors.New("MySQL 접속 설정 형식을 확인하세요")
		}
		db = sql.OpenDB(connector)
	case "mssql":
		cfg := msdsn.Config{Host: host, Port: uint64(number(c.Config, "port", 1433)), Database: database, User: user, Password: password, Encryption: msdsn.EncryptionRequired, TLSConfig: tlsConfig, AppName: "madi-readonly-source"}
		if plain {
			cfg.Encryption = msdsn.EncryptionDisabled
			cfg.TLSConfig = nil
		}
		connector := mssql.NewConnectorConfig(cfg)
		connector.Dialer = dial
		db = sql.OpenDB(connector)
	case "oracle":
		options := map[string]string{"CONNECTION TIMEOUT": strconv.Itoa(int(timeout.Seconds())), "TRACE FILE": ""}
		if !plain {
			options["SSL"] = "enable"
			options["SSL VERIFY"] = "true"
		}
		connector := goora.NewConnector(goora.BuildUrl(host, number(c.Config, "port", 1521), database, user, password, options)).(*goora.OracleConnector)
		connector.Dialer(dial)
		if !plain {
			connector.WithTLSConfig(tlsConfig)
		}
		db = sql.OpenDB(connector)
	default:
		return nil, errors.New("지원하지 않는 외부 데이터베이스입니다")
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(time.Minute)
	return db, nil
}

// SQL Server has no read-only transaction mode. Its current effective login and
// object privileges are verified separately, and only our fixed SELECT grammar
// is executable. Other engines additionally enforce read-only on the transaction.
func beginSQLSource(ctx context.Context, db *sql.DB, kind string) (*sql.Tx, error) {
	tx, e := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: kind != "oracle" && kind != "mssql"})
	if e != nil {
		return nil, errors.New("외부 DB 읽기 전용 트랜잭션을 시작하지 못했습니다")
	}
	if kind == "oracle" {
		if _, e = tx.ExecContext(ctx, "SET TRANSACTION READ ONLY"); e != nil {
			_ = tx.Rollback()
			return nil, errors.New("Oracle 읽기 전용 트랜잭션을 설정하지 못했습니다")
		}
	}
	return tx, nil
}
func sqlParam(kind string, n int) string {
	switch kind {
	case "postgres":
		return "$" + strconv.Itoa(n)
	case "oracle":
		return ":" + strconv.Itoa(n)
	case "mssql":
		return "@p" + strconv.Itoa(n)
	default:
		return "?"
	}
}
func sqlIdent(kind, name string) string {
	switch kind {
	case "mysql", "mariadb":
		return "`" + name + "`"
	case "mssql":
		return "[" + name + "]"
	default:
		return `"` + name + `"`
	}
}
func checkSQLSourcePrivileges(ctx context.Context, tx *sql.Tx, c sqlSource, schema, table string) error {
	fail := errors.New("원격 계정은 해당 테이블의 읽기 전용 최소 권한 계정이어야 합니다")
	switch c.Kind {
	case "postgres":
		var unsafe bool
		e := tx.QueryRowContext(ctx, `SELECT r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR has_table_privilege(current_user,$1,'INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') FROM pg_roles r WHERE r.rolname=current_user`, sqlIdent(c.Kind, schema)+"."+sqlIdent(c.Kind, table)).Scan(&unsafe)
		if e != nil || unsafe {
			return fail
		}
	case "mysql", "mariadb":
		rows, e := tx.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER")
		if e != nil {
			return fail
		}
		defer rows.Close()
		for rows.Next() {
			var grant string
			if rows.Scan(&grant) != nil {
				return fail
			}
			upper := strings.ToUpper(grant)
			at := strings.Index(upper, " ON ")
			if !strings.HasPrefix(upper, "GRANT ") || at < 6 || strings.Contains(upper, "WITH GRANT OPTION") {
				return fail
			}
			for _, priv := range strings.Split(upper[6:at], ",") {
				if !oneOf(strings.TrimSpace(priv), "SELECT", "USAGE", "SHOW VIEW") {
					return fail
				}
			}
		}
		if rows.Err() != nil {
			return fail
		}
	case "mssql":
		var admin int
		if tx.QueryRowContext(ctx, "SELECT CASE WHEN IS_SRVROLEMEMBER('sysadmin')=1 OR IS_MEMBER('db_owner')=1 THEN 1 ELSE 0 END").Scan(&admin) != nil || admin != 0 {
			return fail
		}
		object := sqlIdent(c.Kind, schema) + "." + sqlIdent(c.Kind, table)
		rows, e := tx.QueryContext(ctx, "SELECT permission_name FROM fn_my_permissions(@p1,'OBJECT')", object)
		if e != nil {
			return fail
		}
		defer rows.Close()
		for rows.Next() {
			var privilege string
			if rows.Scan(&privilege) != nil || !oneOf(privilege, "SELECT", "VIEW DEFINITION") {
				return fail
			}
		}
		if rows.Err() != nil {
			return fail
		}
	case "oracle":
		var user string
		if tx.QueryRowContext(ctx, "SELECT USER FROM dual").Scan(&user) != nil || strings.EqualFold(user, schema) {
			return fail
		}
		var unsafe int
		e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM SESSION_PRIVS WHERE PRIVILEGE NOT IN ('CREATE SESSION','SELECT ANY TABLE','SELECT ANY DICTIONARY')`).Scan(&unsafe)
		if e != nil || unsafe > 0 {
			return fail
		}
		e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ALL_TAB_PRIVS WHERE TABLE_SCHEMA=:1 AND TABLE_NAME=:2 AND PRIVILEGE<>'SELECT' AND (GRANTEE=USER OR GRANTEE='PUBLIC' OR GRANTEE IN (SELECT ROLE FROM SESSION_ROLES))`, schema, table).Scan(&unsafe)
		if e != nil || unsafe > 0 {
			return fail
		}
	}
	return nil
}

type sqlColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

func inspectSQLTable(ctx context.Context, tx *sql.Tx, c sqlSource, schema, table string) ([]sqlColumn, error) {
	if e := checkSQLSourcePrivileges(ctx, tx, c, schema, table); e != nil {
		return nil, e
	}
	query := "SELECT column_name,data_type,is_nullable FROM information_schema.columns WHERE table_schema=" + sqlParam(c.Kind, 1) + " AND table_name=" + sqlParam(c.Kind, 2) + " ORDER BY ordinal_position"
	if c.Kind == "oracle" {
		query = "SELECT COLUMN_NAME,DATA_TYPE,NULLABLE FROM ALL_TAB_COLUMNS WHERE OWNER=:1 AND TABLE_NAME=:2 ORDER BY COLUMN_ID"
	}
	rows, e := tx.QueryContext(ctx, query, schema, table)
	if e != nil {
		return nil, errors.New("외부 테이블 메타데이터를 조회하지 못했습니다")
	}
	defer rows.Close()
	columns := []sqlColumn{}
	for rows.Next() {
		var c sqlColumn
		var nullable string
		if e = rows.Scan(&c.Name, &c.Type, &nullable); e != nil {
			return nil, errors.New("외부 컬럼 형식을 읽지 못했습니다")
		}
		c.Nullable = nullable == "YES" || nullable == "Y"
		if len(columns) >= 200 {
			return nil, errors.New("테이블 컬럼은 최대 200개입니다")
		}
		columns = append(columns, c)
	}
	if rows.Err() != nil || len(columns) == 0 {
		return nil, fmt.Errorf("테이블 %s.%s의 읽기 권한 또는 존재 여부를 확인하세요", schema, table)
	}
	return columns, nil
}
