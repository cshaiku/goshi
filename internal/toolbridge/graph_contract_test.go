package toolbridge

import (
	"embed"
	"encoding/json"
	"reflect"
	"testing"
)

//go:embed graph_contract_*.json
var graphContracts embed.FS

func TestGraphNativeProtocolContracts(t *testing.T) {
	for _, kind := range []string{"workflow", "plan", "run", "verification"} {
		t.Run(kind, func(t *testing.T) {
			data, err := graphContracts.ReadFile("graph_contract_" + kind + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var want, got any
			if err = json.Unmarshal(data, &want); err != nil {
				t.Fatal(err)
			}
			schema, err := graphSchema(kind)
			if err != nil {
				t.Fatal(err)
			}
			current, err := json.Marshal(schema)
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(current, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatal("protocol contract changed: review and version the golden schema")
			}
		})
	}
}
