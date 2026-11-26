package torrent

import (
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// QbitClient is a client for qBittorrent's WebAPI (v2)
type QbitClient struct {
	BaseURL    string
	Username   string
	Password   string
	httpClient *http.Client
	mu         sync.Mutex
	cookie     string
}

// QbitTorrent represents a torrent in qBittorrent
type QbitTorrent struct {
	Hash          string  `json:"hash"`
	Name          string  `json:"name"`
	State         string  `json:"state"`    // downloading, uploading, stalledUP, stalledDL, pausedUP, pausedDL, error
	Progress      float64 `json:"progress"` // 0.0 to 1.0
	Size          int64   `json:"size"`
	Downloaded    int64   `json:"downloaded"`
	DownloadSpeed int64   `json:"dlspeed"`
	UploadSpeed   int64   `json:"upspeed"`
	Seeders       int     `json:"num_seeds"`
	Peers         int     `json:"num_leechs"`
	SavePath      string  `json:"save_path"`
	ContentPath   string  `json:"content_path"`
	Category      string  `json:"category"`
	Tags          string  `json:"tags"`
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
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.loginLocked()
}

func (q *QbitClient) loginLocked() error {
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
		return fmt.Errorf("qbit login failed (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	// Extract session cookie (if auth is bypassed on localhost, qBit may omit SID)
	for _, c := range resp.Cookies() {
		if c.Name == "SID" {
			q.cookie = c.Name + "=" + c.Value
			return nil
		}
	}
	q.cookie = "SID="
	return nil
}

// doRequest performs an authenticated HTTP request to qBittorrent, re-authenticating once on 401/403
func (q *QbitClient) doRequest(method, path string, bodyStr string, contentType string) (*http.Response, error) {
	q.mu.Lock()
	if q.cookie == "" {
		if err := q.loginLocked(); err != nil {
			q.mu.Unlock()
			return nil, err
		}
	}
	cookie := q.cookie
	q.mu.Unlock()

	makeReq := func(ck string) (*http.Response, error) {
		var bodyReader io.Reader
		if bodyStr != "" {
			bodyReader = strings.NewReader(bodyStr)
		}
		req, err := http.NewRequest(method, q.BaseURL+path, bodyReader)
		if err != nil {
			return nil, err
		}
		if ck != "" && ck != "SID=" {
			req.Header.Set("Cookie", ck)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		return q.httpClient.Do(req)
	}

	resp, err := makeReq(cookie)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		q.mu.Lock()
		err = q.loginLocked()
		newCookie := q.cookie
		q.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return makeReq(newCookie)
	}
	return resp, nil
}

// AddMagnet adds a magnet link to qBittorrent and optionally sets a save path
func (q *QbitClient) AddMagnet(magnetURL, savePath string) error {
	return q.AddMagnetWithCategory(magnetURL, savePath, "")
}

// AddMagnetWithCategory adds a magnet link with an explicit save path and optional category.
func (q *QbitClient) AddMagnetWithCategory(magnetURL, savePath, category string) error {
	formData := url.Values{}
	formData.Set("urls", magnetURL)
	formData.Set("autoTMM", "false")
	if savePath != "" {
		formData.Set("savepath", savePath)
	}
	if category != "" {
		formData.Set("category", category)
	}
	formData.Set("sequentialDownload", "false")

	resp, err := q.doRequest("POST", "/api/v2/torrents/add",
		formData.Encode(),
		"application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("qbit add magnet request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusOK || bodyStr != "Ok." {
		return fmt.Errorf("qbit add magnet failed (HTTP %d): %s", resp.StatusCode, bodyStr)
	}
	return nil
}

// GetTorrents returns all torrents from qBittorrent
func (q *QbitClient) GetTorrents() ([]QbitTorrent, error) {
	resp, err := q.doRequest("GET", "/api/v2/torrents/info", "", "")
	if err != nil {
		return nil, fmt.Errorf("qbit get torrents request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qbit get torrents HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

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
	hash = strings.ToLower(strings.TrimSpace(hash))
	for i := range torrents {
		if strings.ToLower(torrents[i].Hash) == hash {
			return &torrents[i], nil
		}
	}
	return nil, fmt.Errorf("torrent not found: %s", hash)
}

// DeleteTorrent removes a torrent from qBittorrent
func (q *QbitClient) DeleteTorrent(hash string, deleteFiles bool) error {
	deleteFilesStr := "false"
	if deleteFiles {
		deleteFilesStr = "true"
	}

	formData := url.Values{}
	formData.Set("hashes", strings.ToLower(strings.TrimSpace(hash)))
	formData.Set("deleteFiles", deleteFilesStr)

	resp, err := q.doRequest("POST", "/api/v2/torrents/delete",
		formData.Encode(),
		"application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("qbit delete torrent request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbit delete torrent failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// PauseTorrent pauses a torrent by hash (supports qBit v4 /pause and v5 /stop)
func (q *QbitClient) PauseTorrent(hash string) error {
	formData := url.Values{}
	formData.Set("hashes", strings.ToLower(strings.TrimSpace(hash)))
	payload := formData.Encode()

	resp, err := q.doRequest("POST", "/api/v2/torrents/pause", payload, "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("qbit pause request failed: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		resp, err = q.doRequest("POST", "/api/v2/torrents/stop", payload, "application/x-www-form-urlencoded")
		if err != nil {
			return fmt.Errorf("qbit stop request failed: %w", err)
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbit pause failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ResumeTorrent resumes a torrent by hash (supports qBit v4 /resume and v5 /start)
func (q *QbitClient) ResumeTorrent(hash string) error {
	formData := url.Values{}
	formData.Set("hashes", strings.ToLower(strings.TrimSpace(hash)))
	payload := formData.Encode()

	resp, err := q.doRequest("POST", "/api/v2/torrents/resume", payload, "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("qbit resume request failed: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		resp, err = q.doRequest("POST", "/api/v2/torrents/start", payload, "application/x-www-form-urlencoded")
		if err != nil {
			return fmt.Errorf("qbit start request failed: %w", err)
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbit resume failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// TestConnection tests if the qBittorrent WebAPI is reachable and credentials are correct
func (q *QbitClient) TestConnection() error {
	q.mu.Lock()
	q.cookie = ""
	q.mu.Unlock()
	return q.Login()
}

// ExtractMagnetHash extracts the lowercase 40-char hex infohash from a magnet URI.
// Supports both hex (40 chars) and base32 (32 chars) btih values.
func ExtractMagnetHash(magnetURI string) string {
	parsed, err := url.Parse(magnetURI)
	if err != nil {
		return ""
	}
	for _, xt := range parsed.Query()["xt"] {
		const prefix = "urn:btih:"
		if strings.HasPrefix(strings.ToLower(xt), prefix) {
			raw := strings.TrimSpace(xt[len(prefix):])
			if len(raw) == 40 {
				return strings.ToLower(raw)
			}
			if len(raw) == 32 {
				if decoded, err := base32.StdEncoding.DecodeString(strings.ToUpper(raw)); err == nil {
					return strings.ToLower(hex.EncodeToString(decoded))
				}
			}
			return strings.ToLower(raw)
		}
	}
	return ""
}

// FormatQbitETA formats qBittorrent ETA seconds into a human-readable string.
func FormatQbitETA(seconds int64) string {
	if seconds < 0 || seconds >= 8640000 {
		return "∞"
	}
	if seconds == 0 {
		return "0s"
	}
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// QbitStateToHomeFlix maps qBittorrent torrent state to HomeFlix status string
func QbitStateToHomeFlix(state string) string {
	switch state {
	case "downloading", "metaDL", "forcedMetaDL", "checkingDL", "stalledDL", "queuedDL", "forcedDL", "allocating", "checkingResumeData", "moving":
		return "downloading"
	case "uploading", "stalledUP", "forcedUP", "checkingUP", "queuedUP", "pausedUP", "stoppedUP":
		return "completed"
	case "pausedDL", "stoppedDL":
		return "paused"
	case "error", "missingFiles":
		return "error"
	default:
		return "downloading"
	}
}
