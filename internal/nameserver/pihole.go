package nameserver

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/n6g7/bingo/internal/config"
	"github.com/n6g7/nomtail/pkg/log"
)

type PiholeNS struct {
	logger   *log.Logger
	baseURL  string
	password string
	client   *JsonClient
}

func NewPiholeNS(logger *log.Logger, conf config.PiholeConf) *PiholeNS {
	return &PiholeNS{
		logger:   logger.With("component", "pi-hole"),
		baseURL:  conf.URL,
		password: conf.Password,
	}
}

// Send an HTTP request to the Pi-hole, with built-in CSRF token refresh.
func (ph *PiholeNS) do(method, uri string, reqBody, respBody any) error {
	req, err := http.NewRequest(method, ph.baseURL+uri, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	err = ph.client.DoJSON(req, reqBody, respBody)
	if err != nil {
		// If we get a 401, it probably means the CSRF token has expired. Login again to refresh it.
		if strings.Contains(err.Error(), "status 401 Unauthorized") {
			if err := ph.login(); err != nil {
				return fmt.Errorf("failed to login while refreshing CSRF token: %w", err)
			}
			// Try again.
			return ph.do(method, uri, reqBody, respBody)
		}
		return err
	}
	return nil
}

type loginRequest struct {
	Password string  `json:"password"`
	Totp     *string `json:"totp"`
}
type loginResponse struct {
	Session struct {
		Valid bool   `json:"valid"`
		Csrf  string `json:"csrf"`
	} `json:"session"`
}

func (ph *PiholeNS) login() error {
	var response loginResponse
	err := ph.do(
		"POST",
		"/api/auth",
		&loginRequest{Password: ph.password},
		&response,
	)
	if err != nil {
		return fmt.Errorf("pi-hole login failed: %w", err)
	}

	if !response.Session.Valid {
		return fmt.Errorf("pi-hole login failed due to invalid session (?)")
	}

	ph.client.CsrfToken = response.Session.Csrf

	ph.logger.Debug("pi-hole login successful")
	return nil
}

func (ph *PiholeNS) Init() error {
	// Create HTTP client
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Ignore invalid certs
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("cookie jar creation failed: %w", err)
	}
	ph.client = &JsonClient{Client: http.Client{
		Transport: transport,
		Jar:       jar,
	}}

	// Initial login
	return ph.login()
}

type ListResult struct {
	Config struct {
		DNS struct {
			CNAMERecords struct {
				Value []string `json:"value"`
			} `json:"cnameRecords"`
		} `json:"dns"`
	} `json:"config"`
}

func (ph *PiholeNS) ListRecords() ([]Record, error) {
	output := &ListResult{}
	err := ph.do("GET", "/api/config/dns/cnameRecords?detailed=true", nil, output)
	if err != nil {
		return nil, err
	}

	records := []Record{}
	for _, row := range output.Config.DNS.CNAMERecords.Value {
		items := strings.Split(row, ",")
		records = append(records, Record{items[0], items[1]})
	}
	return records, nil
}

func (ph *PiholeNS) AddRecord(name, cname string) error {
	url := fmt.Sprintf("/api/config/dns/cnameRecords/%s%%2C%s", name, cname)
	return ph.do("PUT", url, nil, nil)
}

func (ph *PiholeNS) RemoveRecord(name string) error {
	// We need to know the target domain in order to delete ...
	records, err := ph.ListRecords()
	if err != nil {
		return err
	}
	target := ""
	for _, record := range records {
		if record.Name == name {
			target = record.Cname
			break
		}
	}
	if target == "" {
		return fmt.Errorf("couldn't find target for domain %s", name)
	}

	url := fmt.Sprintf("/api/config/dns/cnameRecords/%s%%2C%s", name, target)
	return ph.do("DELETE", url, nil, nil)
}
