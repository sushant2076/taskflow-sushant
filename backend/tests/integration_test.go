package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var baseURL string

func TestMain(m *testing.M) {
	baseURL = os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// Wait for API to be ready
	ready := false
	for i := 0; i < 30; i++ {
		resp, err := http.Get(baseURL + "/projects")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				ready = true
				break
			}
		}
		time.Sleep(time.Second)
	}
	if !ready {
		fmt.Println("API not reachable at", baseURL)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// --- helpers ---

func postJSON(t *testing.T, url string, body any, token string) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func getJSON(t *testing.T, url string, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func patchJSON(t *testing.T, url string, body any, token string) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func deleteReq(t *testing.T, url string, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("failed to parse response body: %s", string(b))
	}
	return result
}

func requireStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", want, resp.StatusCode, string(b))
	}
}

// --- Auth Tests ---

func TestAuthFlow(t *testing.T) {
	ts := fmt.Sprintf("%d", time.Now().UnixNano())

	t.Run("register new user", func(t *testing.T) {
		resp := postJSON(t, baseURL+"/auth/register", map[string]string{
			"name":     "Integration Test User",
			"email":    "inttest_" + ts + "@example.com",
			"password": "testpass123",
		}, "")
		requireStatus(t, resp, http.StatusCreated)

		body := readBody(t, resp)
		if _, ok := body["token"]; !ok {
			t.Fatal("expected token in response")
		}
		user := body["user"].(map[string]any)
		if user["email"] != "inttest_"+ts+"@example.com" {
			t.Fatal("email mismatch")
		}
		if _, ok := user["password"]; ok {
			t.Fatal("password should not be in response")
		}
	})

	t.Run("register duplicate email", func(t *testing.T) {
		resp := postJSON(t, baseURL+"/auth/register", map[string]string{
			"name":     "Duplicate",
			"email":    "inttest_" + ts + "@example.com",
			"password": "testpass123",
		}, "")
		requireStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("register missing fields", func(t *testing.T) {
		resp := postJSON(t, baseURL+"/auth/register", map[string]string{}, "")
		requireStatus(t, resp, http.StatusBadRequest)
		body := readBody(t, resp)
		fields := body["fields"].(map[string]any)
		if _, ok := fields["name"]; !ok {
			t.Fatal("expected name validation error")
		}
		if _, ok := fields["email"]; !ok {
			t.Fatal("expected email validation error")
		}
		if _, ok := fields["password"]; !ok {
			t.Fatal("expected password validation error")
		}
	})

	t.Run("login with seed user", func(t *testing.T) {
		resp := postJSON(t, baseURL+"/auth/login", map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}, "")
		requireStatus(t, resp, http.StatusOK)
		body := readBody(t, resp)
		if _, ok := body["token"]; !ok {
			t.Fatal("expected token in response")
		}
	})

	t.Run("login wrong password", func(t *testing.T) {
		resp := postJSON(t, baseURL+"/auth/login", map[string]string{
			"email":    "test@example.com",
			"password": "wrongpassword",
		}, "")
		requireStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("login nonexistent user", func(t *testing.T) {
		resp := postJSON(t, baseURL+"/auth/login", map[string]string{
			"email":    "nobody@example.com",
			"password": "password123",
		}, "")
		requireStatus(t, resp, http.StatusUnauthorized)
	})
}

// --- Auth Middleware Tests ---

func TestAuthMiddleware(t *testing.T) {
	t.Run("no auth header returns 401", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects", "")
		requireStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects", "invalid-token")
		requireStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})
}

// --- Request ID Tests ---

func TestRequestID(t *testing.T) {
	t.Run("response includes X-Request-ID", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects", "")
		defer resp.Body.Close()
		if resp.Header.Get("X-Request-ID") == "" {
			t.Fatal("expected X-Request-ID header in response")
		}
	})

	t.Run("echoes back provided X-Request-ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+"/projects", nil)
		req.Header.Set("X-Request-ID", "test-request-123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.Header.Get("X-Request-ID") != "test-request-123" {
			t.Fatalf("expected X-Request-ID to be echoed, got %s", resp.Header.Get("X-Request-ID"))
		}
	})
}

// --- Project CRUD Tests ---

func TestProjectCRUD(t *testing.T) {
	// Login as seed user
	resp := postJSON(t, baseURL+"/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}, "")
	requireStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	token := body["token"].(string)

	var createdProjectID string

	t.Run("create project", func(t *testing.T) {
		desc := "Integration test project"
		resp := postJSON(t, baseURL+"/projects", map[string]any{
			"name":        "Test Project",
			"description": desc,
		}, token)
		requireStatus(t, resp, http.StatusCreated)
		body := readBody(t, resp)
		createdProjectID = body["id"].(string)
		if body["name"] != "Test Project" {
			t.Fatal("name mismatch")
		}
	})

	t.Run("list projects returns created project", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects", token)
		requireStatus(t, resp, http.StatusOK)
		body := readBody(t, resp)
		projects := body["projects"].([]any)
		if len(projects) == 0 {
			t.Fatal("expected at least one project")
		}
	})

	t.Run("list projects has pagination headers", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects?page=1&limit=5", token)
		requireStatus(t, resp, http.StatusOK)
		if resp.Header.Get("X-Total-Count") == "" {
			t.Fatal("expected X-Total-Count header")
		}
		if resp.Header.Get("X-Page") != "1" {
			t.Fatal("expected X-Page header to be 1")
		}
		if resp.Header.Get("X-Per-Page") != "5" {
			t.Fatal("expected X-Per-Page header to be 5")
		}
		resp.Body.Close()
	})

	t.Run("get project by ID", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects/"+createdProjectID, token)
		requireStatus(t, resp, http.StatusOK)
		body := readBody(t, resp)
		if body["id"] != createdProjectID {
			t.Fatal("id mismatch")
		}
		if _, ok := body["tasks"]; !ok {
			t.Fatal("expected tasks array in response")
		}
	})

	t.Run("get project not found", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects/00000000-0000-0000-0000-000000000000", token)
		requireStatus(t, resp, http.StatusNotFound)
		resp.Body.Close()
	})

	t.Run("update project", func(t *testing.T) {
		resp := patchJSON(t, baseURL+"/projects/"+createdProjectID, map[string]string{
			"name": "Updated Project Name",
		}, token)
		requireStatus(t, resp, http.StatusOK)
		body := readBody(t, resp)
		if body["name"] != "Updated Project Name" {
			t.Fatal("name not updated")
		}
	})

	t.Run("get project stats", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects/"+createdProjectID+"/stats", token)
		requireStatus(t, resp, http.StatusOK)
		body := readBody(t, resp)
		if _, ok := body["total_tasks"]; !ok {
			t.Fatal("expected total_tasks in response")
		}
		if _, ok := body["by_status"]; !ok {
			t.Fatal("expected by_status in response")
		}
		if _, ok := body["by_assignee"]; !ok {
			t.Fatal("expected by_assignee in response")
		}
	})

	t.Run("delete project", func(t *testing.T) {
		resp := deleteReq(t, baseURL+"/projects/"+createdProjectID, token)
		requireStatus(t, resp, http.StatusNoContent)
		resp.Body.Close()
	})

	t.Run("get deleted project returns 404", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects/"+createdProjectID, token)
		requireStatus(t, resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

// --- Task Permission Tests ---

func TestTaskPermissions(t *testing.T) {
	ts := fmt.Sprintf("%d", time.Now().UnixNano())

	// Register user A
	resp := postJSON(t, baseURL+"/auth/register", map[string]string{
		"name":     "User A",
		"email":    "usera_" + ts + "@example.com",
		"password": "testpass123",
	}, "")
	requireStatus(t, resp, http.StatusCreated)
	bodyA := readBody(t, resp)
	tokenA := bodyA["token"].(string)

	// Register user B
	resp = postJSON(t, baseURL+"/auth/register", map[string]string{
		"name":     "User B",
		"email":    "userb_" + ts + "@example.com",
		"password": "testpass123",
	}, "")
	requireStatus(t, resp, http.StatusCreated)
	bodyB := readBody(t, resp)
	tokenB := bodyB["token"].(string)

	// User A creates a project
	resp = postJSON(t, baseURL+"/projects", map[string]any{
		"name": "User A Project",
	}, tokenA)
	requireStatus(t, resp, http.StatusCreated)
	projectBody := readBody(t, resp)
	projectID := projectBody["id"].(string)

	// User A creates a task
	resp = postJSON(t, baseURL+"/projects/"+projectID+"/tasks", map[string]any{
		"title":    "User A Task",
		"priority": "high",
	}, tokenA)
	requireStatus(t, resp, http.StatusCreated)
	taskBody := readBody(t, resp)
	taskID := taskBody["id"].(string)

	t.Run("owner can update project", func(t *testing.T) {
		resp := patchJSON(t, baseURL+"/projects/"+projectID, map[string]string{
			"name": "Updated by A",
		}, tokenA)
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("non-owner cannot update project", func(t *testing.T) {
		resp := patchJSON(t, baseURL+"/projects/"+projectID, map[string]string{
			"name": "Updated by B",
		}, tokenB)
		requireStatus(t, resp, http.StatusForbidden)
		resp.Body.Close()
	})

	t.Run("non-owner cannot delete project", func(t *testing.T) {
		resp := deleteReq(t, baseURL+"/projects/"+projectID, tokenB)
		requireStatus(t, resp, http.StatusForbidden)
		resp.Body.Close()
	})

	t.Run("non-owner/non-creator cannot delete task", func(t *testing.T) {
		resp := deleteReq(t, baseURL+"/tasks/"+taskID, tokenB)
		requireStatus(t, resp, http.StatusForbidden)
		resp.Body.Close()
	})

	t.Run("creator can delete task", func(t *testing.T) {
		resp := deleteReq(t, baseURL+"/tasks/"+taskID, tokenA)
		requireStatus(t, resp, http.StatusNoContent)
		resp.Body.Close()
	})

	t.Run("task list has pagination headers", func(t *testing.T) {
		resp := getJSON(t, baseURL+"/projects/"+projectID+"/tasks?page=1&limit=10", tokenA)
		requireStatus(t, resp, http.StatusOK)
		if resp.Header.Get("X-Total-Count") == "" {
			t.Fatal("expected X-Total-Count header")
		}
		resp.Body.Close()
	})

	// Cleanup
	deleteReq(t, baseURL+"/projects/"+projectID, tokenA)
}
