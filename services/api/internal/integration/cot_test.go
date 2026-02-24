package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestCoT_IngestSingleEvent(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	cotXML := `<event version="2.0" uid="TEST-INGEST-1" type="a-f-G-U-C" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
  <point lat="-33.8688" lon="151.2093" hae="50.0" ce="10.0" le="5.0"/>
  <detail>
    <contact callsign="Ingest1"/>
  </detail>
</event>`

	resp := e.Do(t, "POST", "/api/v1/cot/events", "application/xml", strings.NewReader(cotXML), tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var result struct {
		Total  int `json:"total"`
		Stored int `json:"stored"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
	if result.Stored != 1 {
		t.Errorf("stored = %d, want 1", result.Stored)
	}
}

func TestCoT_IngestBatchEvents(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	cotXML := `<events>
  <event version="2.0" uid="BATCH-1" type="a-f-G-U-C" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
    <point lat="-33.8688" lon="151.2093" hae="0" ce="0" le="0"/>
    <detail/>
  </event>
  <event version="2.0" uid="BATCH-2" type="a-f-G-U-C" time="2024-01-15T10:31:00Z" start="2024-01-15T10:31:00Z" stale="2024-01-15T10:36:00Z" how="m-g">
    <point lat="-33.8700" lon="151.2100" hae="0" ce="0" le="0"/>
    <detail/>
  </event>
</events>`

	resp := e.Do(t, "POST", "/api/v1/cot/events", "application/xml", strings.NewReader(cotXML), tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var result struct {
		Total  int `json:"total"`
		Stored int `json:"stored"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
}

func TestCoT_ListEvents(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Ingest an event first
	cotXML := `<event version="2.0" uid="LIST-TEST-1" type="a-f-G-U-C" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
  <point lat="-33.8688" lon="151.2093" hae="0" ce="0" le="0"/>
  <detail/>
</event>`
	ingestResp := e.Do(t, "POST", "/api/v1/cot/events", "application/xml", strings.NewReader(cotXML), tokens.AccessToken)
	ingestResp.Body.Close()

	// List
	resp := e.DoJSON(t, "GET", "/api/v1/cot/events?page=1&page_size=50", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestCoT_GetLatestByUID(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	uid := "UID-LATEST-TEST-1"
	cotXML := `<event version="2.0" uid="` + uid + `" type="a-f-G-U-C" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
  <point lat="-33.8688" lon="151.2093" hae="0" ce="0" le="0"/>
  <detail><contact callsign="LatestTest"/></detail>
</event>`
	ingestResp := e.Do(t, "POST", "/api/v1/cot/events", "application/xml", strings.NewReader(cotXML), tokens.AccessToken)
	ingestResp.Body.Close()

	// Get latest by UID
	resp := e.DoJSON(t, "GET", "/api/v1/cot/events/"+uid, nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var event struct {
		EventUID string `json:"event_uid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if event.EventUID != uid {
		t.Errorf("event_uid = %q, want %q", event.EventUID, uid)
	}
}

func TestCoT_InvalidXML(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.Do(t, "POST", "/api/v1/cot/events", "application/xml", strings.NewReader("<not valid xml"), tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusBadRequest)
}

func TestCoT_Unauthenticated(t *testing.T) {
	e := getEnv(t)
	resp := e.DoJSON(t, "GET", "/api/v1/cot/events", nil, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusUnauthorized)
}
