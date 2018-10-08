package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	resty "gopkg.in/resty.v1"
)

// {
// 	"auth": {
// 	  "identity": {
// 		"methods": [
// 		  "password"
// 		],
// 		"password": {
// 		  "user": {
// 			"domain": {
// 			  "name": "$OS_USER_DOMAIN_NAME"
// 			},
// 			"name": "$OS_USERNAME",
// 			"password": "$OS_PASSWORD"
// 		  }
// 		}
// 	  },
// 	  "scope": {
// 		"project": {
// 		  "domain": {
// 			"name": "$OS_PROJECT_DOMAIN_NAME"
// 		  },
// 		  "name": "$OS_PROJECT_NAME"
// 		}
// 	  }
// 	}
//   }

type domain struct {
	// ID   *string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	// Domain   domain `json:"domain"`
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

// type testme struct {
// 	Nested struct {
// 		Name string `json:"name"`
// 	} `json:"nested"`
// }

type AuthSuccess struct {
	Token struct {
		AuditIDs []string `json:"audit_ids"`
		Catalog  *[]struct {
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
		} `json:"catalog"`
		ExpiresAt string   `json:"expires_at"`
		IsDomain  bool     `json:"is_domain"`
		IssuedAt  string   `json:"issued_at"`
		Methods   []string `json:"methods"`
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

	var computeURL *string

	for _, catalog := range *(resp.Result().(*AuthSuccess).Token.Catalog) {
		if catalog.Name == "nova" {
			for _, endpoint := range catalog.Endpoints {
				if endpoint.Interface == "public" {
					computeURL = new(string)
					*computeURL = endpoint.URL
				}
			}
		}
	}

	if computeURL == nil {
		fmt.Printf("Error: unable to determine compute URL\n")

		os.Exit(1)
	}

	fmt.Printf("Compute URL: %s\n", *computeURL)

	// fmt.Printf("%v\n", resp.Header()["X-Subject-Token"])

	subjectToken := resp.Header()["X-Subject-Token"][0]

	flavorURL := *computeURL + "/flavors"

	resp2, err2 := resty.R().
		SetHeader("X-Auth-Token", subjectToken).
		SetResult(&ComputeFlavorsResponse{}).
		Get(flavorURL)
	if err2 != nil {
		fmt.Printf("Error: %v\n", err2)

	}

	fmt.Printf("Response status code: %v\n", resp2.StatusCode())

	// fmt.Printf("%v\n", resp.Result().(*AuthSuccess).Token.Catalog)

	// fmt.Printf("%v\n", resp2.Result().(*ComputeFlavorsResponse).Flavors)

	for _, flavor := range resp2.Result().(*ComputeFlavorsResponse).Flavors {
		fmt.Printf("%s %s\n", flavor.Name, flavor.ID)
	}
}
