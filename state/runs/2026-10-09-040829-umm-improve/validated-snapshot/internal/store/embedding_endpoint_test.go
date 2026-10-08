package store

import "testing"

// Leaving the embedding address empty must keep behaving exactly as before:
// embeddings go to the chat gateway with its key. Every existing install is in
// this state, and an upgrade that silently stopped embedding would look like the
// model had broken.
func TestEmbeddingFallsBackToTheChatGateway(t *testing.T) {
	settings := embeddingSettings{
		BaseURL: "https://gateway.example", APIKey: "enc:chat-secret", EmbeddingModel: "bge-m3",
	}
	url, key := settings.embeddingEndpoint()
	if url != "https://gateway.example" {
		t.Errorf("base url = %q, want the chat gateway", url)
	}
	if key != "enc:chat-secret" {
		t.Errorf("key = %q, want the chat gateway's", key)
	}
}

// Setting the embedding address sends embeddings there instead.
func TestEmbeddingUsesItsOwnGatewayWhenSet(t *testing.T) {
	settings := embeddingSettings{
		BaseURL: "https://gateway.example", APIKey: "enc:chat-secret",
		EmbeddingBaseURL: "http://embeddings:11434", EmbeddingAPIKey: "enc:embed-secret",
		EmbeddingModel: "bge-m3",
	}
	url, key := settings.embeddingEndpoint()
	if url != "http://embeddings:11434" {
		t.Errorf("base url = %q, want the embedding gateway", url)
	}
	if key != "enc:embed-secret" {
		t.Errorf("key = %q, want the embedding gateway's", key)
	}
}

// The rule that makes splitting the address safe.
//
// A key issued for one host is a credential for that host. Carrying it over
// because the embedding key was left blank would hand it to whoever runs the
// other machine — and an embedding server with no authentication is the ordinary
// case, so blank has to mean "send nothing", not "send the other one".
func TestTheChatKeyNeverTravelsToTheEmbeddingHost(t *testing.T) {
	settings := embeddingSettings{
		BaseURL: "https://gateway.example", APIKey: "enc:chat-secret",
		EmbeddingBaseURL: "http://embeddings:11434",
		EmbeddingModel:   "bge-m3",
	}
	url, key := settings.embeddingEndpoint()
	if url != "http://embeddings:11434" {
		t.Fatalf("base url = %q", url)
	}
	if key != "" {
		t.Errorf("the chat gateway's key was sent to a different host: %q", key)
	}
}

// Whitespace is not an address. Treating it as one would point embeddings at
// nothing while looking configured.
func TestBlankEmbeddingAddressIsNotAnAddress(t *testing.T) {
	settings := embeddingSettings{
		BaseURL: "https://gateway.example", APIKey: "k",
		EmbeddingBaseURL: "   ", EmbeddingModel: "bge-m3",
	}
	url, key := settings.embeddingEndpoint()
	if url != "https://gateway.example" || key != "k" {
		t.Errorf("blank embedding address did not fall back: url=%q key=%q", url, key)
	}
}
