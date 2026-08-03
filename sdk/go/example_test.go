package openlicensd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	openlicensd "github.com/alvarorg14/openlicensd/sdk/go"
)

func ExampleNew() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(openlicensd.ValidationResult{Valid: true})
	}))
	defer server.Close()

	client, err := openlicensd.New(server.URL, "acme-widget")
	if err != nil {
		panic(err)
	}

	result, err := client.Validate(context.Background(), "01234-56789-ABCDE-FGHJK-MNPQR")
	if err != nil {
		panic(err)
	}

	fmt.Println(result.Valid)
	// Output: true
}

func ExampleNormalizeKey() {
	fmt.Println(openlicensd.NormalizeKey("0123456789abcdefghjkmnpqr"))
	// Output: 01234-56789-ABCDE-FGHJK-MNPQR
}
