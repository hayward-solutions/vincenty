package integration

import (
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestAuditLogs_GetMyLogs(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// The login itself should generate an audit log entry.
	resp := e.DoJSON(t, "GET", "/api/v1/audit-logs/me?page=1&page_size=50", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestAuditLogs_AdminGetAll(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", "/api/v1/audit-logs?page=1&page_size=50", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestAuditLogs_NonAdminCannotGetAll(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "auditreguser", "auditreg@test.local", "Password123!", false)
	userTokens := e.Login(t, "auditreguser", "Password123!")

	resp := e.DoJSON(t, "GET", "/api/v1/audit-logs", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}

func TestAuditLogs_ExportMyLogs(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", "/api/v1/audit-logs/me/export?format=json", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestAuditLogs_AdminExportAll(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", "/api/v1/audit-logs/export?format=csv", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestAuditLogs_GroupLogs(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	userID := e.CreateUser(t, adminTokens.AccessToken, "auditgroupuser", "auditgroup@test.local", "Password123!", false)
	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Audit Log Group")
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "admin")

	userTokens := e.Login(t, "auditgroupuser", "Password123!")

	resp := e.DoJSON(t, "GET", "/api/v1/groups/"+groupID.String()+"/audit-logs?page=1&page_size=50", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}
