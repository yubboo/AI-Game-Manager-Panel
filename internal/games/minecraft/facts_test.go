package minecraft

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolverPaperUsesLiveMojangJavaAndPaperSHA(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mc/game/version_manifest_v2.json":
			fmt.Fprintf(w, `{"latest":{"release":"1.21.1"},"versions":[{"id":"1.21.1","type":"release","url":%q}]}`, base+"/version.json")
		case "/version.json":
			fmt.Fprint(w, `{"javaVersion":{"majorVersion":21},"downloads":{"server":{"url":"https://example/server.jar","sha1":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":123}}}`)
		case "/projects/paper/versions/1.21.1/builds/latest":
			fmt.Fprint(w, `{"id":133,"downloads":{"server:default":{"name":"paper.jar","url":"https://example/paper.jar","size":456,"checksums":{"sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL
	facts, err := (Resolver{Client: srv.Client(), MojangBase: base, PaperBase: base, FabricBase: base}).Resolve(context.Background(), "", SoftwarePaper)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Version != "1.21.1" || facts.JavaMajor != 21 || facts.Artifact.Build != "133" || facts.Artifact.HashAlgorithm != "sha256" {
		t.Fatalf("unexpected facts: %#v", facts)
	}
}
