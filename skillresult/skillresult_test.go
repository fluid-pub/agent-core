package skillresult

import "testing"

func TestSuccess(t *testing.T) {
	m := Success(map[string]interface{}{"zones": []string{"Z1"}})
	if m[KeyOK] != true {
		t.Fatalf("ok: got %v", m[KeyOK])
	}
	data, ok := m[KeyData].(map[string]interface{})
	if !ok || data["zones"] == nil {
		t.Fatalf("data: %#v", m[KeyData])
	}
}

func TestSuccessNilData(t *testing.T) {
	m := Success(nil)
	data, _ := m[KeyData].(map[string]interface{})
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %#v", data)
	}
}

func TestFailure(t *testing.T) {
	m := Failure("boom")
	if m[KeyOK] != false || m[KeyError] != "boom" {
		t.Fatalf("got %#v", m)
	}
}
