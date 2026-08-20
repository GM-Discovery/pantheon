package pantheon

import "testing"

func TestDecodeApolloModelOutputAcceptsFencedJSON(t *testing.T) {
	var output apolloModelOutput
	if err := decodeApolloModelOutput("```json\n{\"answer\":\"A cited answer\",\"status\":\"complete\"}\n```", &output); err != nil {
		t.Fatal(err)
	}
	if output.Answer != "A cited answer" || output.Status != "complete" {
		t.Fatalf("decoded Apollo output = %#v", output)
	}
}

func TestDecodeApolloModelOutputExtractsJSONFromBoundedWrapper(t *testing.T) {
	var output apolloModelOutput
	if err := decodeApolloModelOutput("Here is the requested result:\n{\"answer\":\"bounded\",\"status\":\"complete\"}", &output); err != nil {
		t.Fatal(err)
	}
	if output.Answer != "bounded" {
		t.Fatalf("decoded Apollo answer = %q", output.Answer)
	}
}
