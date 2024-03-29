package nameserver

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/lizongying/go-xpath/xpath"
	"github.com/n6g7/bingo/internal/config"
	"github.com/n6g7/nomtail/pkg/log"
)

type PiholeNS struct {
	logger   *log.Logger
	baseURL  string
	password string
	client   *http.Client
}

func initClient() (*http.Client, error) {
	transport := &http.Transport{
		// Ignore invalid certs
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookie jar creation failed: %w", err)
	}
	client := &http.Client{
		Transport: transport,
		Jar:       jar,
	}
	return client, nil
}

func NewPiholeNS(logger *log.Logger, conf config.PiholeConf) *PiholeNS {
	return &PiholeNS{
		logger:   logger.With("component", "pi-hole"),
		baseURL:  conf.URL,
		password: conf.Password,
	}
}

func (ph *PiholeNS) login() error {
	resp, err := ph.client.PostForm(
		ph.baseURL+"/admin/login.php",
		url.Values{"pw": {ph.password}},
	)
	if err != nil {
		return fmt.Errorf("pi-hole login failed: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("pi-hole login failed")
	}
	ph.logger.Debug("pi-hole login successful")
	return nil
}

func (ph *PiholeNS) Init() error {
	client, err := initClient()
	if err != nil {
		return fmt.Errorf("pi-hole client creation failed: %w", err)
	}
	ph.client = client
	return ph.login()
}

func (ph *PiholeNS) getCSRFToken() (string, error) {
	resp, err := ph.client.Get(ph.baseURL + "/admin/cname_records.php")
	if err != nil {
		return "", err
	}
	xp, err := xpath.NewXpathFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	return xp.FindStrOne(`//*[@id="token"]`), nil
}

func (ph *PiholeNS) request(uri string, qs url.Values, output any) error {
	token, err := ph.getCSRFToken()
	if err != nil {
		return fmt.Errorf("error fetching CSRF token: %w", err)
	}
	qs.Add("token", token)

	resp, err := ph.client.PostForm(ph.baseURL+uri, qs)
	if err != nil {
		return fmt.Errorf("pi-hole request to '%s' failed: %w", uri, err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code while querying %s: %d", uri, resp.StatusCode)
	}

	// Attempt parsing body
	rawBody := &bytes.Buffer{}
	body := io.TeeReader(resp.Body, rawBody)
	decoder := json.NewDecoder(body)
	err = decoder.Decode(output)
	if err != nil {
		// Error parsing response body, are we logged out?
		if rawBody.String() == "Session expired! Please re-login on the Pi-hole dashboard." {
			// Login + re-attemp request
			ph.login()
			return ph.request(uri, qs, output)
		}

		return fmt.Errorf("error parsing '%s' response: %w", uri, err)
	}
	return nil
}

type ListResult struct {
	Data [][]string `json:"data"`
}

func (ph *PiholeNS) ListRecords() ([]Record, error) {
	output := &ListResult{}
	err := ph.request(
		"/admin/scripts/pi-hole/php/customcname.php",
		url.Values{"action": {"get"}},
		output,
	)
	if err != nil {
		return nil, err
	}

	records := []Record{}
	for _, row := range output.Data {
		records = append(records, Record{row[0], row[1]})
	}
	return records, nil
}

type GenericResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (ph *PiholeNS) AddRecord(name, cname string) error {
	output := &GenericResult{}
	err := ph.request(
		"/admin/scripts/pi-hole/php/customcname.php",
		url.Values{
			"action": {"add"},
			"domain": {name},
			"target": {cname},
		},
		output,
	)
	if err != nil {
		return err
	}

	if !output.Success {
		return fmt.Errorf("error while creating record for '%s': %s", name, output.Message)
	}

	return nil
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

	output := &GenericResult{}
	err = ph.request(
		"/admin/scripts/pi-hole/php/customcname.php",
		url.Values{
			"action": {"delete"},
			"domain": {name},
			"target": {target},
		},
		output,
	)
	if err != nil {
		return err
	}

	if !output.Success {
		return fmt.Errorf("error while deleting record for '%s': %s", name, output.Message)
	}

	return nil
}
