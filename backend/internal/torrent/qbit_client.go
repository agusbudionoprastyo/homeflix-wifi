package torrent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// QbitClient is a client for qBittorrent's WebAPI (v2)
type QbitClient struct {
	BaseURL    string
	Username   string
	Password   string
	httpClient *http.Client
	cookie     string
}

// QbitTorrent represents a torrent in qBittorrent
type QbitTorrent struct {
	Hash          string  `json:"hash"`
	Name          string  `json:"name"`
	State         string  `json:"state"` // downloading, uploading, stalledUP, stalledDL, pausedUP, pausedDL, error
	Progress      float64 `json:"progress"`
	Size          int64   `json:"size"`
	Downloaded    int64   `json:"downloaded"`
	DownloadSpeed int64   `json:"dlspeed"`
	UploadSpeed   int64   `json:"upspeed"`
	Seeders       int     `json:"num_seeds"`
	Peers         int     `json:"num_leechs"`
	SavePath      string  `json:"save_path"`
	CompletedOn   int64   `json:"completion_on"` // Unix timestamp, -1 if not completed
	AddedOn       int64   `json:"added_on"`
	ETA           int64   `json:"eta"` // seconds, 8640000 means unknown
}

// NewQbitClient creates a new qBittorrent client
func NewQbitClient(baseURL, username, password string) *QbitClient {
	return &QbitClient{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Username: username,
		Password: password,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Login authenticates with qBittorrent and stores the session cookie
func (q *QbitClient) Login() error {
	formData := url.Values{}
	formData.Set("username", q.Username)
	formData.Set("password", q.Password)

	resp, err := q.httpClient.PostForm(q.BaseURL+"/api/v2/auth/login", formData)
	if err != nil {
		return fmt.Errorf("qbit login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("qbit login read body failed: %w", err)
	}

	bodyStr := strings.TrimSpace(string(body))
	if bodyStr != "Ok." {
		return fmt.Errorf("qbit login failed: %s", bodyStr)
	}

	// Extract session cookie
	for _, c := range resp.Cookies() {
		if c.Name == "SID" {
			q.cookie = c.Name + "=" + c.Value
			return nil
		}
	}
	return fmt.Errorf("qbit login: no SID cookie in response")
}

// doRequest performs an authenticated HTTP request to qBittorrent
func (q *QbitClient) doRequest(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, q.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if q.cookie != "" {
		req.Header.Set("Cookie", q.cookie)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return q.httpClient.Do(req)
}

// ensureLoggedIn logs in if we don't have a session cookie yet
func (q *QbitClient) ensureLoggedIn() error {
	if q.cookie != "" {
		return nil
	}
	return q.Login()
}

// AddMagnet adds a magnet link to qBittorrent and optionally sets a save path
func (q *QbitClient) AddMagnet(magnetURL, savePath string) error {
	if err := q.ensureLoggedIn(); err != nil {
		return err
	}

	formData := url.Values{}
	formData.Set("urls", magnetURL)
	if savePath != "" {
		formData.Set("savepath", savePath)
	}
	formData.Set("sequentialDownload", "false")

	resp, err := q.doRequest("POST", "/api/v2/torrents/add",
		strings.NewReader(formData.Encode()),
		"application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("qbit add magnet request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(body))
	if bodyStr != "Ok." {
		return fmt.Errorf("qbit add magnet failed: %s", bodyStr)
	}
	return nil
}

// GetTorrents returns all torrents from qBittorrent
func (q *QbitClient) GetTorrents() ([]QbitTorrent, error) {
	if err := q.ensureLoggedIn(); err != nil {
		return nil, err
	}

	resp, err := q.doRequest("GET", "/api/v2/torrents/info", nil, "")
	if err != nil {
		return nil, fmt.Errorf("qbit get torrents request failed: %w", err)
	}
	defer resp.Body.Close()

	var torrents []QbitTorrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, fmt.Errorf("qbit get torrents decode failed: %w", err)
	}
	return torrents, nil
}

// GetTorrent returns info about a single torrent by hash
func (q *QbitClient) GetTorrent(hash string) (*QbitTorrent, error) {
	torrents, err := q.GetTorrents()
	if err != nil {
		return nil, err
	}
	hash = strings.ToLower(hash)
	for _, t := range torrents {
		if strings.ToLower(t.Hash) == hash {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("torrent not found: %s", hash)
}

// DeleteTorrent removes a torrent from qBittorrent
func (q *QbitClient) DeleteTorrent(hash string, deleteFiles bool) error {
	if err := q.ensureLoggedIn(); err != nil {
		return err
	}

	deleteFilesStr := "false"
	if deleteFiles {
		deleteFilesStr = "true"
	}

	formData := url.Values{}
	formData.Set("hashes", strings.ToLower(hash))
	formData.Set("deleteFiles", deleteFilesStr)

	resp, err := q.doRequest("POST", "/api/v2/torrents/delete",
		strings.NewReader(formData.Encode()),
		"application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("qbit delete torrent request failed: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// PauseTorrent pauses a torrent by hash
func (q *QbitClient) PauseTorrent(hash string) error {
	if err := q.ensureLoggedIn(); err != nil {
		return err
	}
	formData := url.Values{}
	formData.Set("hashes", strings.ToLower(hash))
	resp, err := q.doRequest("POST", "/api/v2/torrents/pause",
		strings.NewReader(formData.Encode()),
		"application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("qbit pause request failed: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// ResumeTorrent resumes a torrent by hash
func (q *QbitClient) ResumeTorrent(hash string) error {
	if err := q.ensureLoggedIn(); err != nil {
		return err
	}
	formData := url.Values{}
	formData.Set("hashes", strings.ToLower(hash))
	resp, err := q.doRequest("POST", "/api/v2/torrents/resume",
		strings.NewReader(formData.Encode()),
		"application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("qbit resume request failed: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// TestConnection tests if the qBittorrent WebAPI is reachable and credentials are correct
func (q *QbitClient) TestConnection() error {
	// Reset cookie to force fresh login
	q.cookie = ""
	return q.Login()
}

// QbitStateToHomeFlix maps qBittorrent torrent state to HomeFlix status string
func QbitStateToHomeFlix(state string) string {
	switch state {
	case "downloading", "metaDL", "checkingDL":
		return "downloading"
	case "uploading", "stalledUP", "forcedUP", "checkingUP":
		return "completed"
	case "pausedDL", "stalledDL":
		return "paused"
	case "pausedUP":
		return "completed"
	case "error", "missingFiles", "unknown":
		return "error"
	default:
		return "downloading"
	}
}
