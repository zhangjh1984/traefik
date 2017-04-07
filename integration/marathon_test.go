package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/containous/traefik/integration/utils"
	marathon "github.com/gambol99/go-marathon"
	"github.com/go-check/check"

	checker "github.com/vdemeester/shakers"
)

// Marathon test suites (using libcompose)
type MarathonSuite struct{ BaseSuite }

func (s *MarathonSuite) SetUpSuite(c *check.C) {
	fmt.Println("setting up marathon compose file")
	s.createComposeProject(c, "marathon")
	s.composeProject.Start(c)
}

func (s *MarathonSuite) TestSimpleConfiguration(c *check.C) {
	cmd := exec.Command(traefikBinary, "--configFile=fixtures/marathon/simple.toml")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	var resp *http.Response
	// TODO validate : run on 80
	err = utils.TryRequest("http://127.0.0.1:8000/", 5*time.Second, func(res *http.Response) error {
		resp = res
		return nil
	})
	c.Assert(err, checker.IsNil)
	defer resp.Body.Close()

	// Expected a 404 as we did not configure anything
	c.Assert(resp.StatusCode, checker.Equals, 404)
}

func (s *MarathonSuite) TestConfigurationUpdate(c *check.C) {
	cmd := exec.Command(traefikBinary, "--configFile=fixtures/marathon/with-entrypoint.toml")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	marathonURL := "http://marathon:8080"
	fmt.Printf("polling Marathon URL %s for availability\n", marathonURL)
	// wait for marathon
	err = utils.TryRequest(fmt.Sprintf("%s/ping", marathonURL), 360*time.Second, func(res *http.Response) error {
		res.Body.Close()
		return nil
	})
	c.Assert(err, checker.IsNil)

	// Prepare Marathon client.
	config := marathon.NewDefaultConfig()
	config.URL = marathonURL
	client, err := marathon.NewClient(config)
	c.Assert(err, checker.IsNil)

	// Deploy test application via Marathon.
	app := marathon.NewDockerApplication().Name("/whoami").CPU(0.1).Memory(32)
	app.Container.Docker.Container("emilevauge/whoami")

	fmt.Println("deploying test application")
	deployID, err := client.UpdateApplication(app, false)
	c.Assert(err, checker.IsNil)
	c.Assert(client.WaitOnDeployment(deployID.DeploymentID, 30*time.Second), checker.IsNil)

	fmt.Println("done.")
}
