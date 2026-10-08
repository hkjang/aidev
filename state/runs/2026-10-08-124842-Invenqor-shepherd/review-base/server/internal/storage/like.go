package storage

import "strings"

// Search text typed by an operator is a literal, but every LIKE built from it
// read it as a pattern. "%" matched every row, "_" matched any character - and
// "_" is ordinary in the names this product stores, so searching the audit log
// for api_key.create also returned api-key.create - and a backslash meant one
// thing to each engine: PostgreSQL takes it as the default LIKE escape while
// the SQLite fallback has no default escape character at all, so a Windows path
// such as C:\Program found rows in one mode and none in the other.
//
// The escape character is "!" because the software inventory search already
// escaped its pattern that way; this is the same rule in one place.
const (
	// LikeEscapeCharacter is the escape character LikePattern inserts.
	LikeEscapeCharacter = "!"
	// LikeEscapeClause must follow every LIKE whose pattern came from
	// LikePattern or LikeContains. It carries no formatting verb, so it is
	// safe to concatenate into a statement built with fmt.Sprintf.
	LikeEscapeClause = " ESCAPE '" + LikeEscapeCharacter + "'"
)

var likeEscaper = strings.NewReplacer(
	LikeEscapeCharacter, LikeEscapeCharacter+LikeEscapeCharacter,
	"%", LikeEscapeCharacter+"%",
	"_", LikeEscapeCharacter+"_",
)

// LikePattern escapes the LIKE metacharacters in value so it matches itself.
func LikePattern(value string) string {
	return likeEscaper.Replace(value)
}

// LikeContains is the substring form: the escaped value between wildcards.
func LikeContains(value string) string {
	return "%" + LikePattern(value) + "%"
}
