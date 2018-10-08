package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	resty "gopkg.in/resty.v1"
)

type domain struct {
	Name string `json:"name"`
}

type User struct {
	Domain struct {
		Name string `json:"name"`
	} `json:"domain"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type password struct {
	User struct {
		Domain struct {
			Name string `json:"name"`
		} `json:"domain"`
		Name     string `json:"name"`
		Password string `json:"password"`
	} `json:"user"`
}

type identity struct {
	Methods  []string `json:"methods"`
	Password password `json:"password"`
}

type projectdomain struct {
	ID   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

type project struct {
	Domain projectdomain `json:"domain"`
	Name   string        `json:"name"`
}

type scope struct {
	Project project `json:"project"`
}

type nested struct {
	Identity identity `json:"identity"`
	Scope    scope    `json:"scope"`
}

type openstackAuthStruct struct {
	Auth nested `json:"auth"`
}

type CatalogStruct struct {
	Endpoints []struct {
		ID        string `json:"id"`
		Interface string `json:"interface"`
		Region    string `json:"region"`
		RegionID  string `json:"region_id"`
		URL       string `json:"url"`
	} `json:"endpoints"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type AuthSuccess struct {
	Token struct {
		AuditIDs  []string         `json:"audit_ids"`
		Catalog   *[]CatalogStruct `json:"catalog"`
		ExpiresAt string           `json:"expires_at"`
		IsDomain  bool             `json:"is_domain"`
		IssuedAt  string           `json:"issued_at"`
		Methods   []string         `json:"methods"`
		Project   struct {
			Domain struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"domain"`
			ID   string `json:"id"`
			Name string `json:"string"`
		} `json:"project"`
		Roles []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"roles"`
		User struct {
			Domain struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			ID                string `json:"id"`
			Name              string `json:"name"`
			PasswordExpiresAt string `json:"password_expires_at"`
		} `json:"user"`
	} `json:"token"`
}

type ComputeFlavorsResponse struct {
	Flavors []struct {
		ID    string `json:"id"`
		Links []struct {
			HREF string `json:"href"`
			Rel  string `json:"rel"`
		}
		Name string `json:"name"`
	} `json:"flavors"`
}

func main() {
	osAuthURL := strings.Trim(os.Getenv("OS_AUTH_URL"), "/")

	if osAuthURL == "" {
		fmt.Printf("Error: environment not set (OS_AUTH_URL)\n")
		os.Exit(1)
	}

	value := os.Getenv("OS_PROJECT_DOMAIN_ID")

	blah := openstackAuthStruct{
		Auth: nested{
			Identity: identity{
				Methods: []string{"password"},
				Password: password{
					User{
						Domain: domain{
							Name: os.Getenv("OS_USER_DOMAIN_NAME"),
						},
						Name:     os.Getenv("OS_USERNAME"),
						Password: os.Getenv("OS_PASSWORD"),
					},
				},
			},
			Scope: scope{
				Project: project{
					Domain: projectdomain{
						ID: &value,
					},
					Name: os.Getenv("OS_PROJECT_NAME"),
				},
			},
		},
	}

	b, err := json.Marshal(blah)
	if err != nil {
		fmt.Println("Error:", err)

		os.Exit(1)
	}

	resty.SetHeader("Accept", "application/json")
	resty.SetHeader("Content-type", "application/json")

	url := osAuthURL + "/auth/tokens"

	fmt.Printf("%v\n", url)

	resp, errblah := resty.R().
		SetBody(b).
		SetResult(&AuthSuccess{}).
		// SetError(&AuthError{}).
		Post(url)
	if errblah != nil {
		fmt.Printf("%s\n", errblah)
	}

	fmt.Printf("Response status code: %v\n", resp.StatusCode())

	// Use the following snippet to parse the timestamps returned by OpenStack
	// fmt.Println(time.Parse(time.RFC3339, resp.Result().(*AuthSuccess).Token.ExpiresAt))

	computeURL := getPublicComputeURL(resp.Result().(*AuthSuccess).Token.Catalog)
	if computeURL == nil {
		fmt.Printf("Error: unable to determine compute URL\n")

		os.Exit(1)
	}

	fmt.Printf("Compute URL: %s\n", *computeURL)

	subjectToken := resp.Header()["X-Subject-Token"][0]

	flavorURL := *computeURL + "/flavors"

	resp2, err2 := resty.R().
		SetHeader("X-Auth-Token", subjectToken).
		SetResult(&ComputeFlavorsResponse{}).
		Get(flavorURL)
	if err2 != nil {
		fmt.Printf("Error: %v\n", err2)

		os.Exit(1)
	}

	if resp2.StatusCode() != http.StatusOK {
		fmt.Printf("Error: HTTP status %d\n", resp2.StatusCode())

		os.Exit(1)
	}

	for _, flavor := range resp2.Result().(*ComputeFlavorsResponse).Flavors {
		fmt.Printf("%s %s\n", flavor.Name, flavor.ID)
	}
}

func getPublicComputeURL(catalog *[]CatalogStruct) *string {
	return getURLFromCatalog(catalog, "nova", "public")
}

func getURLFromCatalog(catalog *[]CatalogStruct, name string, intfc string) *string {
	var computeURL *string

	for _, catalog := range *catalog {
		if catalog.Name != name {
			continue
		}

		for _, endpoint := range catalog.Endpoints {
			if endpoint.Interface == intfc {
				computeURL = new(string)
				*computeURL = endpoint.URL

				break
			}
		}

		if computeURL != nil {
			break
		}
	}

	return computeURL
}
