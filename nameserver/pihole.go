package nameserver

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/lizongying/go-xpath/xpath"
	"github.com/n6g7/bingo/config"
)

type PiholeNS struct {
	baseURL       string
	password      string
	client        *http.Client
	serviceDomain string
}

func initClient() (*http.Client, error) {
	transport := &http.Transport{
		// Ignore invalid certs
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("Cookie jar creation failed: %w", err)
	}
	client := &http.Client{
		Transport: transport,
		Jar:       jar,
	}
	return client, nil
}

func NewPiholeNS(conf config.PiholeConf, serviceDomain string) (*PiholeNS, error) {
	client, err := initClient()
	if err != nil {
		return nil, fmt.Errorf("Pihole client creation failed: %w", err)
	}
	return &PiholeNS{
		conf.URL,
		conf.Password,
		client,
		serviceDomain,
	}, nil
}

func (ph *PiholeNS) login() error {
	resp, err := ph.client.PostForm(
		ph.baseURL+"/admin/login.php",
		url.Values{"pw": {ph.password}},
	)
	if err != nil {
		return fmt.Errorf("Pihole login failed: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Pi-hole login failed")
	}
	return nil
}

func (ph *PiholeNS) Init() error {
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
		return fmt.Errorf("Error fetching CSRF token: %w", err)
	}
	qs.Add("token", token)

	resp, err := ph.client.PostForm(ph.baseURL+uri, qs)
	if err != nil {
		return fmt.Errorf("Pihole request to '%s' failed: %w", uri, err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Error while querying %s: %d", uri, resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(output)
	if err != nil {
		return fmt.Errorf("Error parsing '%s' response: %w", uri, err)
	}
	return nil
}

type ListResult struct {
	Data [][]string `json:"data"`
}

func (ph *PiholeNS) isServiceDomain(domain string) bool {
	return strings.HasSuffix(domain, ph.serviceDomain)
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
		if ph.isServiceDomain(row[0]) {
			records = append(records, Record{row[0], row[1]})
		}
	}
	return records, nil
}

type GenericResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (ph *PiholeNS) AddRecord(name, cname string) error {
	if !ph.isServiceDomain(name) {
		return fmt.Errorf("%s isn't a service domain", name)
	}

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
		return fmt.Errorf("Error while creating record for '%s': %s", name, output.Message)
	}

	return nil
}

func (ph *PiholeNS) RemoveRecord(name string) error {
	if !ph.isServiceDomain(name) {
		return fmt.Errorf("%s isn't a service domain", name)
	}

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
		return fmt.Errorf("Couldn't find target for domain %s", name)
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
		return fmt.Errorf("Error while deleting record for '%s': %s", name, output.Message)
	}

	return nil
}
