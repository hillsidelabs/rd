package infra

import (
	"encoding/json"
	"os/exec"
)

type tfoutput struct {
	Values struct {
		RootModule struct {
			Resources []struct {
				Values struct {
					Name      string `json:"name"`
					IP        string `json:"ipv4_address"`
					PrivateIP string `json:"ipv4_address_private"`
				} `json:"values"`
			} `json:"resources"`
		} `json:"root_module"`
	} `json:"values"`
}

func GetHostFromDOTerraform() ([]Host, error) {
	cmd := exec.Command("terraform", "show", "-json")
	cmd.Dir = "./infra"
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var tf tfoutput
	if err := json.Unmarshal(out, &tf); err != nil {
		return nil, err
	}

	hosts := []Host{}

	for _, value := range tf.Values.RootModule.Resources {
		// Only add a host if it has an address.
		if value.Values.IP != "" && value.Values.PrivateIP != "" {
			hosts = append(hosts, Host{
				Name:      value.Values.Name,
				IP:        value.Values.IP,
				PrivateIP: value.Values.PrivateIP,
			})
		}
	}

	return hosts, nil
}
