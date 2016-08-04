package stackplan

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"koding/db/models"
	"koding/kites/kloud/contexthelper/session"
	"koding/kites/kloud/klient"
	"koding/kites/kloud/utils"

	"github.com/hashicorp/terraform/terraform"
	"github.com/koding/kite"
	"github.com/koding/logging"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2/bson"
)

var defaultLog = logging.NewCustom("stackplan", false)

// credPermissions defines the permission grid for the given method
var credPermissions = map[string][]string{
	"bootstrap":    []string{"owner"},
	"plan":         []string{"user", "owner"},
	"apply":        []string{"user", "owner"},
	"authenticate": []string{"user", "owner"},
	"migrate":      []string{"owner"},
}

// Machine represents a jComputeStack.machine value.
type Machine struct {
	Provider        string            `json:"provider"`
	Label           string            `json:"label"`
	Region          string            `json:"region"`
	QueryString     string            `json:"queryString,omitempty"`
	RegisterURL     string            `json:"registerURL,omitempty"`
	HostQueryString string            `json:"hostQueryString,omitempty"`
	Attributes      map[string]string `json:"attributes"`
}

// DialState describes state of a single dial.
type DialState struct {
	Label   string // the vm label
	KiteID  string // the kite ID being checked
	KiteURL string // last resolved URL or empty
	State   string // either "kontrol", "dial", "ping" or "provider"
	Err     error  // underlying error
}

// DialError describes an error of CheckKlients.
type DialError struct {
	States []*DialState
}

// Error implements the built-in error interface.
func (de *DialError) Error() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "Failed to dial the following kites:\n\n")

	for _, s := range de.States {
		fmt.Fprintf(&buf, "  * %s: %s (id=%q, url=%q, state=%q)\n", s.Label, s.Err, s.KiteID, s.KiteURL, s.State)
	}

	return buf.String()
}

// Err returns de if it contains at least 1 failed state.
func (de *DialError) Err() error {
	for _, s := range de.States {
		if s.Err != nil {
			return de
		}
	}

	return nil
}

// KiteMap maps resource names to kite IDs they own.
type KiteMap map[string]string

// Stack is struct that contains all necessary information Apply needs to
// perform successfully.
type Stack struct {
	ID bson.ObjectId // jComputeStack._id

	// Machines is a list of jMachine identifiers.
	Machines []string

	// Credentials maps jCredential provider to identifiers.
	Credentials map[string][]string

	// Template is a raw Terraform template.
	Template string

	// Stack is a jComputeStack value.
	Stack *models.ComputeStack
}

// Machines is a list of machines.
type Machines struct {
	Machines []Machine `json:"machines"`
}

// Credential represents jCredential{Datas} value. Meta is of a provider-specific
// type, defined by a ctor func in MetaFuncs map.
type Credential struct {
	Title      string
	Provider   string
	Identifier string
	Meta       interface{}
}

// String implememts the fmt.Stringer interface.
func (m *Machines) String() string {
	var txt string
	for i, machine := range m.Machines {
		txt += fmt.Sprintf("[%d] %+v\n", i, machine)
	}
	return txt
}

func (m *Machines) AppendRegion(region string) {
	for i, machine := range m.Machines {
		machine.Region = region
		m.Machines[i] = machine
	}
}

func (m *Machines) AppendQueryString(queryStrings map[string]string) {
	for i, machine := range m.Machines {
		queryString, ok := queryStrings[machine.Label]
		if !ok {
			continue
		}

		machine.QueryString = utils.QueryString(queryString)
		m.Machines[i] = machine
	}
}

func (m *Machines) AppendRegisterURL(urls map[string]string) {
	for i, machine := range m.Machines {
		registerURL, ok := urls[machine.QueryString]
		if !ok {
			continue
		}

		machine.RegisterURL = registerURL
		m.Machines[i] = machine
	}
}

func (m *Machines) AppendHostQueryString(s string) {
	for i, machine := range m.Machines {
		machine.HostQueryString = utils.QueryString(s)
		m.Machines[i] = machine
	}
}

// WithLabel returns the machine with the associated label
func (m *Machines) WithLabel(label string) (Machine, error) {
	for _, machine := range m.Machines {
		if machine.Label == label {
			return machine, nil
		}
	}

	return Machine{}, fmt.Errorf("couldn't find machine with label '%s", label)
}

// UserData injects header/footer into custom script and ensures it has
// a shebang line.
func UserData(content string) string {
	var buf bytes.Buffer

	// If there's no shebang, execute the script with sh.
	if !strings.HasPrefix(content, "#!") {
		fmt.Fprintln(&buf, "#!/bin/sh")
	}

	fmt.Fprintln(&buf, content)

	return buf.String()
}

// DefaultKlientTimeout specifies the maximum time we're going to try to
// connect to klient before timing out.
var DefaultKlientTimeout = 5 * time.Minute

// Planner is used to build kloud machines from Terraform resources,
// like plan result or state file.
//
// It is also used for checking connectivity with klient running
// on those machines.
type Planner struct {
	Provider     string // Terraform provider name
	ResourceType string // Terraform resource type

	KlientTimeout time.Duration // when zero-value, DefaultKlientTimeout is used

	// OnKlient, when non-nil, is called to perform additional check
	// for CheckKlients method.
	OnKlient func(*kite.Client) error

	// SessionFunc is used to build a session value from the context.
	//
	// When nil, session.FromContext is used by default.
	SessionFunc func(context.Context) (*session.Session, error)
}

// MachinesFromState builds a list of machines from Terraform state value.
//
// It ignores any other resources than those specified by p.ResourceType
// and p.Provider.
func (p *Planner) MachinesFromState(state *terraform.State) (*Machines, error) {
	if len(state.Modules) == 0 {
		return nil, errors.New("state modules is empty")
	}

	var out Machines

	for _, m := range state.Modules {
		for resource, r := range m.Resources {
			if r.Primary == nil {
				continue
			}

			provider, resourceType, label, err := parseResource(resource)
			if err != nil {
				return nil, err
			}

			if resourceType != p.ResourceType || provider != p.Provider {
				continue
			}

			attrs := make(map[string]string, len(r.Primary.Attributes))
			for key, val := range r.Primary.Attributes {
				attrs[key] = val
			}

			out.Machines = append(out.Machines, Machine{
				Provider:   provider,
				Label:      label,
				Attributes: attrs,
			})
		}
	}

	return &out, nil
}

// MachinesFromPlan builds a list of machines from Terraform plan result.
//
// It ignores any other resources than those specified by p.ResourceType
// and p.Provider.
func (p *Planner) MachinesFromPlan(plan *terraform.Plan) (*Machines, error) {
	if plan.Diff == nil {
		return nil, errors.New("plan diff is empty")
	}

	if len(plan.Diff.Modules) == 0 {
		return nil, errors.New("plan diff module is empty")
	}

	var out Machines

	for _, d := range plan.Diff.Modules {
		if d.Resources == nil {
			continue
		}

		for providerResource, r := range d.Resources {
			if len(r.Attributes) == 0 {
				continue
			}

			provider, resourceType, label, err := parseResource(providerResource)
			if err != nil {
				return nil, err
			}

			if resourceType != p.ResourceType || provider != p.Provider {
				continue
			}

			attrs := make(map[string]string, len(r.Attributes))
			for name, a := range r.Attributes {
				attrs[name] = a.New
			}

			out.Machines = append(out.Machines, Machine{
				Provider:   provider,
				Label:      label,
				Attributes: attrs,
			})
		}
	}

	return &out, nil
}

func (p *Planner) checkSingleKlient(k *kite.Kite, label, kiteID string) *DialState {
	kiteID = utils.QueryString(kiteID)

	start := time.Now()

	c, err := klient.NewWithTimeout(k, kiteID, p.klientTimeout())
	if err == klient.ErrDialingFailed {
		return &DialState{
			Label:   label,
			KiteID:  kiteID,
			KiteURL: c.Client.URL,
			State:   "dial",
			Err:     err,
		}
	}

	if err != nil {
		return &DialState{
			Label:  label,
			KiteID: kiteID,
			State:  "kontrol",
			Err:    err,
		}
	}

	defer c.Close()

	left := p.klientTimeout() - time.Now().Sub(start)

	err = c.PingTimeout(max(left, klient.DefaultTimeout))
	if err != nil {
		return &DialState{
			Label:  label,
			KiteID: kiteID,
			State:  "ping",
			Err:    err,
		}
	}

	if p.OnKlient != nil {
		err = p.OnKlient(c.Client)
	}

	return &DialState{
		Label:  label,
		KiteID: kiteID,
		State:  "provider",
		Err:    err,
	}
}

// CheckKlients checks connectivity to all klient kites given by the kiteIDs
// parameter.
//
// It returns RegisterURLs mapped to each kite's query string.
func (p *Planner) CheckKlients(ctx context.Context, kiteIDs KiteMap) (map[string]string, error) {
	sess, err := p.session(ctx)
	if err != nil {
		return nil, err
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex // protects multierror and outputs
		urls = make(map[string]string, len(kiteIDs))
		de   = &DialError{}
	)

	for l, k := range kiteIDs {
		wg.Add(1)

		go func(label, kiteID string) {
			defer wg.Done()
			state := p.checkSingleKlient(sess.Kite, label, kiteID)

			mu.Lock()
			de.States = append(de.States, state)
			urls[state.KiteID] = state.KiteURL
			mu.Unlock()
		}(l, k)
	}

	wg.Wait()

	return urls, de.Err()
}

func (p *Planner) klientTimeout() time.Duration {
	if p.KlientTimeout != 0 {
		return p.KlientTimeout
	}
	return DefaultKlientTimeout
}

func (p *Planner) session(ctx context.Context) (*session.Session, error) {
	if p.SessionFunc != nil {
		return p.SessionFunc(ctx)
	}

	sess, ok := session.FromContext(ctx)
	if !ok {
		return nil, errors.New("session context is not passed")
	}

	return sess, nil
}

func parseResource(resource string) (string, string, string, error) {
	// resource is in the form of "aws_instance.foo.bar"
	splitted := strings.SplitN(resource, "_", 2)
	if len(splitted) < 2 {
		return "", "", "", fmt.Errorf("provider resource is unknown: %v", splitted)
	}

	resourceSplitted := strings.SplitN(splitted[1], ".", 2)

	provider := splitted[0]             // aws
	resourceType := resourceSplitted[0] // instance
	label := resourceSplitted[1]        // foo.bar

	return provider, resourceType, label, nil
}

// isVariable checkes whether the given string is a template variable, such as:
// "${var.region}"
func IsVariable(v string) bool {
	return len(v) != 0 && v[0] == '$'
}

// FlattenValues converts the values of a map[string][]string to a []string slice.
func FlattenValues(kv map[string][]string) []string {
	values := []string{}

	for _, val := range kv {
		values = append(values, val...)
	}

	return values
}

func max(d, t time.Duration) time.Duration {
	if d > t {
		return d
	}
	return t
}
