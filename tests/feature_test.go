// +build acceptance

package tests

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	"github.com/DATA-DOG/godog/gherkin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	yaml "gopkg.in/yaml.v2"
)

var opt = godog.Options{
	Paths:  []string{"."},
	Output: colors.Colored(os.Stdout),
	// Format: "pretty",
	Format:      "progress",
	Concurrency: 4,
	Randomize:   time.Now().UTC().UnixNano(),
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)
	rand.Seed(time.Now().UnixNano())
}

func TestMain(m *testing.M) {
	flag.Parse()

	status := godog.RunWithOptions("stas", func(s *godog.Suite) {
		FeatureContext(s)
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}

	os.Exit(status)
}

type feature struct {
	scenarioID string
	testDir    string
	featurID   string

	lastOutput string
	lastErr    error

	cf *cloudformation.CloudFormation
}

func (f *feature) fileExists(fname string, content *gherkin.DocString) error {
	fpath := filepath.Join(f.testDir, fname)
	dir, _ := filepath.Split(fpath)

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	c := strings.Replace(content.Content, "%scenarioid%", f.scenarioID, -1)
	c = strings.Replace(c, "%featureid%", f.featurID, -1)

	return ioutil.WriteFile(fpath, []byte(c), 0700)
}

func (f *feature) iSuccessfullyRun(cmd string) error {
	err := f.iRun(cmd)
	if err != nil {
		return err
	}

	if f.lastErr != nil {
		return fmt.Errorf("err: %v, output:\n%s", err, string(f.lastOutput))
	}

	return nil
}

func (f *feature) stackShouldHaveStatus(stackName, status string) error {
	s := strings.Replace(stackName, "%scenarioid%", f.scenarioID, -1)
	out, err := f.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(s),
	})
	if err != nil {
		return err
	}

	if aws.StringValue(out.Stacks[0].StackStatus) != status {
		return fmt.Errorf("stack status is %s", aws.StringValue(out.Stacks[0].StackStatus))
	}

	return nil
}

func (f *feature) iModifyFile(fname string, content *gherkin.DocString) error {
	return f.fileExists(fname, content)
}

func (f *feature) iRun(cmd string) error {
	bin, err := filepath.Abs("../bin/stas")
	if err != nil {
		return err
	}

	c := exec.Command(bin, strings.Split(cmd, " ")...)
	c.Dir = f.testDir

	out, err := c.CombinedOutput()
	f.lastOutput = string(out)
	f.lastErr = err

	return nil
}

func (f *feature) exitCodeShouldNotBeZero() error {
	if f.lastErr == nil {
		return errors.New("program returned zero exit code")
	}
	return nil
}

func (f *feature) outputShouldContain(s *gherkin.DocString) error {
	if !strings.Contains(f.lastOutput, s.Content) {
		return fmt.Errorf("output doesn't contain searched string. Output:\n%s", f.lastOutput)
	}
	return nil
}

func (f *feature) thereShouldBeStackThatMatches(stackName string, expectedContent *gherkin.DocString) error {
	stackName = strings.Replace(stackName, "%scenarioid%", f.scenarioID, -1)
	out, err := f.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return err
	}

	actualStackData := out.Stacks[0]

	expectedStackData := struct {
		StackStatus string
		Resources   map[string]string
	}{}

	c := strings.Replace(expectedContent.Content, "%scenarioid%", f.scenarioID, -1)
	err = yaml.Unmarshal([]byte(c), &expectedStackData)
	if err != nil {
		return err
	}

	if expectedStackData.StackStatus != "" {
		actualStatus := aws.StringValue(actualStackData.StackStatus)
		if actualStatus != expectedStackData.StackStatus {
			return fmt.Errorf("status %s doesn't match status %s of stack %s", expectedStackData.StackStatus, actualStatus, stackName)
		}
	}

	for expectedResKey, expectedResValue := range expectedStackData.Resources {
		actualResource, err := f.cf.DescribeStackResource(&cloudformation.DescribeStackResourceInput{
			StackName:         aws.String(stackName),
			LogicalResourceId: aws.String(expectedResKey),
		})
		if err != nil {
			return err
		}

		s := strings.Split(aws.StringValue(actualResource.StackResourceDetail.PhysicalResourceId), ":")
		actualResValue := s[len(s)-1]

		if actualResValue != expectedResValue {
			return fmt.Errorf("resource %s is expected to have value %s. Actual value: %s", expectedResKey, expectedResValue, actualResValue)
		}
	}

	return nil
}

func FeatureContext(s *godog.Suite) {
	testDir := "./.tmp"
	f := feature{}

	sess := session.Must(session.NewSession())
	cf := cloudformation.New(sess)

	f.cf = cf

	s.Step(`^file "([^"]*)" exists:$`, f.fileExists)
	s.Step(`^I successfully run "([^"]*)"$`, f.iSuccessfullyRun)
	s.Step(`^stack "([^"]*)" should have status "([^"]*)"$`, f.stackShouldHaveStatus)
	s.Step(`^I modify file "([^"]*)":$`, f.iModifyFile)
	s.Step(`^I run "([^"]*)"$`, f.iRun)
	s.Step(`^exit code should not be zero$`, f.exitCodeShouldNotBeZero)
	s.Step(`^output should contain:$`, f.outputShouldContain)
	s.Step(`^there should be stack "([^"]*)" that matches:$`, f.thereShouldBeStackThatMatches)

	s.BeforeScenario(func(interface{}) {
		f.scenarioID = strconv.FormatInt(rand.Int63(), 10)
		f.testDir = filepath.Join(testDir, "stas_test_"+f.scenarioID)
	})
	s.AfterScenario(func(interface{}, error) {
		os.RemoveAll(f.testDir)
	})

	s.BeforeFeature(func(*gherkin.Feature) {
		f.featurID = strconv.FormatInt(rand.Int63(), 10)
	})

	s.AfterFeature(func(*gherkin.Feature) {
		stacks, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{})
		if err != nil {
			panic(err)
		}

		for _, s := range stacks.Stacks {
			for _, t := range s.Tags {
				if aws.StringValue(t.Key) == "STAS_TEST" && aws.StringValue(t.Value) == f.featurID {
					cf.DeleteStack(&cloudformation.DeleteStackInput{
						StackName: s.StackName,
					})
				}
			}
		}
	})
}
