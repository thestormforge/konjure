package spec

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

var schemeOverride = regexp.MustCompile(`^[a-zA-Z](?:[a-zA-Z0-9+\-.])*::`)

type Parser struct {
	Reader io.Reader
}

// Decode converts a string into a resource. The goal here is to be compatible with Kustomize where we overlap,
// (e.g. Git URLs), but there may be additional functionality handled here.
func (p *Parser) Decode(spec string) (interface{}, error) {
	// Default reader
	if spec == "-" {
		if p.Reader != nil {
			return &kio.ByteReader{Reader: p.Reader}, nil
		}
		return nil, nil
	}

	// Absolute file path
	if filepath.IsAbs(spec) {
		return &konjurev1beta2.File{Path: spec}, nil
	}

	// Process scheme overrides
	if so := schemeOverride.FindString(spec); so != "" {
		switch strings.ToLower(so) {
		case "git::":
			return p.parseGitSpec(spec[len(so):])
		case "helm::":
			return p.parseHelmSpec(spec[len(so):])
		default:
			return nil, fmt.Errorf("unknown scheme override: %s", so)
		}
	}

	// Try to detect other valid URLs
	if u, err := ParseURL(spec); err == nil {
		if strings.HasPrefix(u.Path, "github.com/") {
			u.Host = "github.com"
			spec = "https://" + spec
		}

		if u.Scheme == "ssh" || u.User.Username() == "git" || normalizeGitRepositoryURL(u) {
			return p.parseGitSpec(spec)
		} else if u.Scheme == "http" || u.Scheme == "https" {
			return p.parseHTTPSpec(spec)
		} else if u.Scheme == "k8s" {
			return p.parseKubernetesSpec(spec)
		}
	}

	return &konjurev1beta2.File{Path: spec}, nil
}

func (p *Parser) parseGitSpec(spec string) (interface{}, error) {
	u, err := ParseURL(spec)
	if err != nil {
		return nil, err
	}

	// TODO What if it comes back opaque?

	q := u.Query()
	u.RawQuery = ""

	g := &konjurev1beta2.Git{
		Refspec: q.Get("ref"),
		Context: normalizeGitRepositoryPath(u),
	}

	if g.Refspec == "" {
		g.Refspec = q.Get("version")
	}

	normalizeGitRepositoryURL(u)
	g.Repository = u.String()

	return g, nil
}

func (p *Parser) parseHelmSpec(spec string) (interface{}, error) {
	u, err := url.Parse(spec)
	if err != nil {
		return nil, err
	}

	helm := &konjurev1beta2.Helm{}

	// The fragment shouldn't be used for anything so let it be the release name
	helm.ReleaseName = u.Fragment
	u.Fragment = ""

	// None of these URLs should be using query parameters, just make them into values for the chart
	q := u.Query()
	u.RawQuery = ""
	for k, v := range q {
		helm.Values = append(helm.Values, konjurev1beta2.HelmValue{
			Name:  k,
			Value: v[0],
		})
	}

	switch {

	case u.Host == "artifacthub.io" && strings.HasPrefix(u.Path, "/packages/helm/"):
		// If this looks like an Artifact Hub URL, try to pull the details via the API
		if resp, err := http.Get("https://artifacthub.io/api/v1" + u.Path); err == nil && resp.StatusCode == http.StatusOK {
			hpkg := struct {
				Repository struct {
					URL string `json:"url"`
				} `json:"repository"`
				Name    string `json:"name"`
				Version string `json:"version"`
			}{}
			if err := json.NewDecoder(resp.Body).Decode(&hpkg); err == nil {
				helm.Repository = hpkg.Repository.URL
				helm.Chart = hpkg.Name
				helm.Version = hpkg.Version
			}
		}

	case path.Ext(u.Path) == ".tgz":
		// If this looks like an actual chart URL, assume the index is in the same place
		u.Path, helm.Chart = path.Split(strings.TrimSuffix(u.Path, path.Ext(u.Path)))
		helm.Repository = u.String()
		if pos := strings.LastIndexByte(helm.Chart, '-'); pos > 0 {
			helm.Version = helm.Chart[pos+1:]
			helm.Chart = helm.Chart[0:pos]
		}

	}

	if helm.Chart == "" {
		return nil, fmt.Errorf("unable to resolve Helm chart: %s", spec)
	}

	return helm, nil
}

func (p *Parser) parseHTTPSpec(spec string) (interface{}, error) {
	if u, err := url.Parse(spec); err != nil {
		return nil, err
	} else if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unexpected scheme: %s", spec)
	}

	return &konjurev1beta2.HTTP{URL: spec}, nil
}

func (p *Parser) parseKubernetesSpec(spec string) (interface{}, error) {
	u, err := url.Parse(spec)
	if err != nil {
		return nil, err
	} else if u.Scheme != "k8s" {
		return nil, fmt.Errorf("unexpected scheme: %s", spec)
	}

	ns, t := path.Split(u.Opaque)

	k8s := &konjurev1beta2.Kubernetes{}
	k8s.Namespaces = []string{strings.Trim(ns, "/")}
	k8s.Types = []string{t}
	k8s.LabelSelector = u.Query().Get("labelSelector")
	return k8s, nil
}

func normalizeGitRepositoryURL(repo *URL) bool {
	h := strings.ToLower(repo.Hostname())
	switch {

	// GitHub.
	case h == "github.com":
		repo.Host = "github.com"
		if repo.User.Username() == "git" || repo.Scheme == "ssh" {
			repo.User = url.User("git")
		} else {
			repo.Scheme = "https"
		}

	// https://docs.microsoft.com/en-us/azure/devops/repos/git/clone?view=vsts&tabs=visual-studio#clone_url
	case h == "dev.azure.com" || strings.HasSuffix(h, "visualstudio.com"):
		repo.Path = strings.TrimSuffix(repo.Path, ".git")

	// https://docs.aws.amazon.com/codecommit/latest/userguide/regions.html
	case strings.HasPrefix(h, "git-codecommit") && strings.HasSuffix(h, "amazonaws.com"):
		repo.Path = strings.TrimSuffix(repo.Path, ".git")

	// Not a well-known Git URL (use "git::" to force the behavior for HTTP(S) URLs)
	default:
		return strings.Contains(repo.Path, "_git/")
	}

	return true
}

func normalizeGitRepositoryPath(repo *URL) string {
	var rp string

	var trimSlash bool
	if !strings.HasPrefix(repo.Path, "/") {
		repo.Path = "/" + repo.Path
		trimSlash = true
	}

	if pos := strings.Index(repo.Path, "_git/"); pos >= 0 {
		pos += 5
		if ppos := strings.Index(repo.Path[pos:], "/") + pos; ppos > pos {
			rp = repo.Path[ppos+1:]
			repo.Path = repo.Path[0:ppos]
		}
	} else if pos := strings.Index(repo.Path, ".git"); pos >= 0 {
		rp = strings.TrimLeft(repo.Path[pos+4:], "/")
		repo.Path = repo.Path[0 : pos+4]
	} else if pos := strings.Index(repo.Path, "//"); pos >= 0 {
		rp = repo.Path[pos:]
		repo.Path = repo.Path[0:pos] + ".git"
	} else if p := strings.SplitN(repo.Path, "/", 4); len(p) == 4 {
		rp = p[3]
		repo.Path = strings.Join(p[0:3], "/") + ".git"
	}

	if trimSlash {
		repo.Path = strings.TrimPrefix(repo.Path, "/")
	}

	return rp
}

type URL struct {
	url.URL
}

func ParseURL(rawurl string) (*URL, error) {
	// First use normal URL parsing
	u, err := url.Parse(rawurl)
	if err == nil {
		return &URL{URL: *u}, nil
	}

	// Try SCP-like (e.g. `[user@]host.xz:path/to/repo.git/`)
	if p := strings.SplitN(rawurl, ":", 2); len(p) == 2 {
		if scp, err := url.Parse("scp://" + p[0] + "/" + p[1]); err == nil {
			// Remove the slash we injected to make the parser work
			scp.Path = scp.Path[1:]
			return &URL{URL: *scp}, nil
		}
	}

	// Return the original error
	return nil, err
}

func (u *URL) String() string {
	if u.Scheme == "scp" {
		host := u.Hostname()

		if user := u.User.String(); user != "" {
			return user + "@" + host + ":" + u.Path
		}

		return host + ":" + u.Path
	}

	return u.URL.String()
}
