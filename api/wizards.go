package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type wizardSummary struct {
	Name      string          `json:"name"`
	Category  *string         `json:"category"`
	Shortdesc *string         `json:"shortdesc"`
	Longdesc  json.RawMessage `json:"longdesc"`
}
type wizardLink struct {
	Disabled   bool
	Name       string
	Title      string
	Href       string
	Paragraphs []string
	HelpID     string
}
type wizardCategory struct {
	Title   string
	Icon    string
	ID      string
	Wizards []wizardLink
}

func wizardDescription(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var wrapped struct {
		Content string `json:"__content__"`
	}
	_ = json.Unmarshal(raw, &wrapped)
	return wrapped.Content
}

// Match Wizard.wizard_ok and parse_brief, then WizardHelper's grouping.
func wizardCategories(data []byte) ([]wizardCategory, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 4<<20)
	groups := map[string][]wizardLink{}
	workflows := map[string]bool{"60-nfsserver": true, "mariadb": true, "ocfs2-single": true, "webserver": true}
	for scanner.Scan() {
		var summary wizardSummary
		if err := json.Unmarshal(scanner.Bytes(), &summary); err != nil {
			log.Printf("Error listing scripts: %s", scanner.Text())
			continue
		}
		if summary.Name == "" || summary.Shortdesc == nil {
			continue
		}
		category := "basic"
		if summary.Category != nil {
			category = strings.ToLower(strings.TrimSpace(*summary.Category))
			if category == "script" || (category == "wizard" && workflows[summary.Name]) {
				continue
			}
		}
		category = strings.NewReplacer("/", " ", "\\", " ", ":", " ").Replace(category)
		description := strings.ReplaceAll(wizardDescription(summary.Longdesc), "\r\n", "\n")
		paragraphs := []string{}
		for _, paragraph := range strings.Split(description, "\n\n") {
			if strings.TrimSpace(paragraph) != "" {
				paragraphs = append(paragraphs, strings.ReplaceAll(paragraph, "\n", " "))
			}
		}
		title := strings.TrimSpace(*summary.Shortdesc)
		if summary.Name == "libvirt" {
			title = legacyLibvirtConfig.Title
			paragraphs = append([]string{"This legacy wizard requires the obsolete external/libvirt agent and is disabled. Use Fencing for libvirt (fence_virsh) instead."}, paragraphs...)
		}
		disabled := summary.Name == "vmware" || summary.Name == "libvirt"
		if summary.Name == "vmware" {
			title = deprecatedVMwareTitle
			paragraphs = append([]string{"This legacy wizard requires the obsolete external/vcenter agent and is disabled. Use Fencing using vCenter REST API instead."}, paragraphs...)
		}
		groups[category] = append(groups[category], wizardLink{Name: summary.Name, Title: title, Href: "/cib/live/wizards/" + url.PathEscape(summary.Name), Paragraphs: paragraphs, Disabled: disabled})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(groups))
	for category := range groups {
		names = append(names, category)
	}
	sort.Strings(names)
	categories := []wizardCategory{}
	for i, name := range names {
		title := strings.Join(strings.Fields(name), " ")
		words := strings.Fields(title)
		for j, word := range words {
			letters := []rune(word)
			letters[0] = unicode.ToUpper(letters[0])
			words[j] = string(letters)
		}
		title = strings.Join(words, " ")
		if name == "sap" || name == "nfs" {
			title = strings.ToUpper(name)
		}
		if name == "wizard" {
			title = "Legacy Wizards"
		}
		icon := "magic"
		switch name {
		case "database":
			icon = "database"
		case "fencing", "stonith":
			icon = "plug"
		case "file system":
			icon = "hard-drive"
		case "nfs":
			icon = "globe"
		case "server":
			icon = "server"
		case "wizard":
			icon = "list"
		}
		items := groups[name]
		sort.SliceStable(items, func(i, j int) bool { return items[i].Name < items[j].Name })
		expanded := make([]wizardLink, 0, len(items)+2)
		for _, item := range items {
			expanded = append(expanded, item)
			switch item.Name {
			case "libvirt":
				expanded = append(expanded, wizardLink{Name: "fence-virsh", Title: fenceVirshConfig.Title, Href: "/cib/live/wizards/fence-virsh", Paragraphs: []string{"Fence a libvirt guest using fence_virsh over SSH. Configure the hypervisor address, SSH user, and guest domain name or UUID."}})
			case "vmware":
				expanded = append(expanded, wizardLink{Name: vmwareRESTConfig.Name, Title: vmwareRESTConfig.Title, Href: "/cib/live/wizards/fence-vmware-rest", Paragraphs: []string{"Fence VMware virtual machines through the vCenter REST API using fence_vmware_rest. Requires vCenter; standalone ESXi hosts are not supported."}})
			}
		}
		items = expanded
		for j := range items {
			items[j].HelpID = fmt.Sprintf("wizard-help-%d-%d", i, j)
		}
		categories = append(categories, wizardCategory{Title: title, Icon: icon, ID: regexp.MustCompile(` +`).ReplaceAllString(name, "_"), Wizards: items})
	}
	return categories, nil
}

func listWizards(ctx context.Context) ([]byte, error) {
	// Match CrmScript.run: crmsh parses a quoted JSON command document.
	cmd := exec.CommandContext(ctx, "/usr/sbin/crm", "-f", "/dev/stdin")
	cmd.Stdin = strings.NewReader(`script json "[\"list\"]"` + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return out, nil
}

func WizardsHandler(w http.ResponseWriter, r *http.Request) { serveWizards(w, r, listWizards) }
func serveWizards(w http.ResponseWriter, r *http.Request, list func(context.Context) ([]byte, error)) {
	if r.URL.Path != "/cib/live/wizards" && r.URL.Path != "/cib/live/wizards/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	data, err := list(r.Context())
	if err != nil {
		http.Error(w, "Unable to load wizards: "+err.Error(), 500)
		return
	}
	categories, err := wizardCategories(data)
	if err != nil {
		http.Error(w, "Unable to load wizards: "+err.Error(), 500)
		return
	}
	renderTemplate(w, "wizards", map[string]any{"Title": "Use a wizard", "Categories": categories})
}
