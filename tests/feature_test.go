// +build acceptance

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	"github.com/DATA-DOG/godog/gherkin"
	expect "github.com/Netflix/go-expect"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	assembly "github.com/molecule-man/stack-assembly"
	saaws "github.com/molecule-man/stack-assembly/aws"
	"github.com/molecule-man/stack-assembly/aws/mock"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cmd/commands"
	"github.com/molecule-man/stack-assembly/conf"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

var opt = godog.Options{
	Paths:  []string{"."},
	Output: colors.Colored(os.Stdout),
	// Format:      "pretty",
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
	scenarioName string
	scenarioID   string
	testDir      string
	featureID    string

	lastOutput string
	lastErr    error

	console *expect.Console
	lastCmd *exec.Cmd
	cancel  context.CancelFunc
	wg      *sync.WaitGroup

	cf cloudformationiface.CloudFormationAPI
	fs vfs
}

func (f *feature) assertEgual(expected, actual interface{}, msgAndArgs ...interface{}) error {
	result := assertionResult{}
	assert.Equal(&result, expected, actual, msgAndArgs...)
	return result.err
}

func (f *feature) fileExists(fname string, content *gherkin.DocString) error {
	dir, _ := filepath.Split(fname)

	err := f.fs.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	c := f.replaceParameters(content.Content)

	file, err := f.fs.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(c)
	if err != nil {
		return err
	}

	return file.Sync()
}

func (f *feature) iSuccessfullyRun(cmd string) error {
	err := f.iRun(cmd)
	if err != nil {
		return err
	}

	if f.lastErr != nil {
		return fmt.Errorf("err: %v, output:\n%s", err, f.lastOutput)
	}

	return nil
}

func (f *feature) stackShouldHaveStatus(stackName, status string) error {
	s := f.replaceParameters(stackName)
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

func (f *feature) stackShouldNotExist(stackName string) error {
	s := f.replaceParameters(stackName)
	_, err := f.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(s),
	})

	if err == nil {
		return fmt.Errorf("stack %s is not supposed to exist", s)
	}

	if !strings.Contains(err.Error(), "does not exist") {
		return err
	}
	return nil
}

func (f *feature) iModifyFile(fname string, content *gherkin.DocString) error {
	return f.fileExists(fname, content)
}

func (f *feature) iRun(cmd string) error {
	buf := bytes.NewBuffer([]byte{})
	cli := &cli.CLI{
		Reader:  buf,
		Writer:  buf,
		Errorer: buf,
	}

	c := commands.Commands{
		SA:        assembly.New(cli),
		Cli:       cli,
		CfgLoader: conf.NewLoader(f.fs, mock.New(f.scenarioName, f.featureID, f.scenarioID)),
	}
	root := c.RootCmd()
	root.SetArgs(strings.Split(f.replaceParameters(cmd), " "))
	root.SetOutput(buf)

	err := root.Execute()

	f.lastOutput = buf.String()
	f.lastErr = err

	return nil
}

func (f *feature) exitCodeShouldNotBeZero() error {
	if f.lastErr == nil {
		return fmt.Errorf("program returned zero exit code. Programs output: \n%s", f.lastOutput)
	}
	return nil
}

func (f *feature) outputShouldContain(s *gherkin.DocString) error {
	expected := f.replaceParameters(s.Content)
	if !strings.Contains(f.lastOutput, expected) {
		return fmt.Errorf(
			"output doesn't contain searched string:\n%s\nActual output:\n%s",
			expected,
			f.lastOutput)
	}
	return nil
}

func (f *feature) outputShouldBeExactly(s *gherkin.DocString) error {
	if strings.TrimSpace(f.lastOutput) != strings.TrimSpace(f.replaceParameters(s.Content)) {
		return fmt.Errorf("output isn't equal to expected string. Output:\n%s", f.lastOutput)
	}
	return nil
}

func (f *feature) nodeInJsonOutputShouldBe(nodePath string, expectedContent *gherkin.DocString) error {
	var expected interface{}
	c := f.replaceParameters(expectedContent.Content)
	err := json.Unmarshal([]byte(c), &expected)
	if err != nil {
		return err
	}

	var actual interface{}
	c = f.replaceParameters(f.lastOutput)
	err = json.Unmarshal([]byte(c), &actual)
	if err != nil {
		return fmt.Errorf("err: %s, output:\n%s", err, f.lastOutput)
	}

	for _, key := range strings.Split(nodePath, ".") {
		if key == "" {
			continue
		}

		casted, ok := actual.(map[string]interface{})
		if !ok {
			return fmt.Errorf("not able to find key %s in node (which is not map):\n%#v", key, actual)
		}

		node, ok := casted[key]
		if !ok {
			return fmt.Errorf("not able to find key %s in node:\n%s", key, casted)
		}

		actual = node
	}

	return f.assertEgual(expected, actual)
}

func (f *feature) thereShouldBeStackThatMatches(stackName string, expectedContent *gherkin.DocString) error {
	stackName = f.replaceParameters(stackName)
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
		Tags        map[string]string
	}{}

	c := f.replaceParameters(expectedContent.Content)
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
	for expectedTagKey, expectedTagValue := range expectedStackData.Tags {
		actualTagValue := f.tagValue(actualStackData, expectedTagKey)

		if actualTagValue == "" {
			return fmt.Errorf("tag with key %s is not found in stack %s", expectedTagKey, stackName)
		}

		if actualTagValue != expectedTagValue {
			return fmt.Errorf("tag with key %s is expected to have value %s in stack %s. Actual value: %s", expectedTagKey, expectedTagValue, stackName, actualTagValue)
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

func (f *feature) iLaunched(cmdInstruction string) error {
	c, err := expect.NewConsole(expect.WithDefaultTimeout(15 * time.Second))
	if err != nil {
		return err
	}

	cli := &cli.CLI{
		Reader:  c.Tty(),
		Writer:  c.Tty(),
		Errorer: c.Tty(),
	}

	co := commands.Commands{
		SA:        assembly.New(cli),
		Cli:       cli,
		CfgLoader: conf.NewLoader(f.fs, mock.New(f.scenarioName, f.featureID, f.scenarioID)),
	}
	root := co.RootCmd()
	root.SetArgs(strings.Split(f.replaceParameters(cmdInstruction), " "))
	root.SetOutput(c.Tty())

	f.wg = &sync.WaitGroup{}
	f.wg.Add(1)
	go func() {
		f.lastErr = root.Execute()
		f.wg.Done()
	}()
	f.console = c

	return nil
}

func (f *feature) terminalShows(s *gherkin.DocString) error {
	lines := strings.Split(f.replaceParameters(s.Content), "\n")
	for _, l := range lines {
		o, err := f.console.ExpectString(l)
		if err != nil {
			return fmt.Errorf("error: %v, output:\n%s", err, o)
		}
	}

	return nil
}

func (f *feature) iEnter(s string) error {
	_, err := f.console.SendLine(s)
	return err
}

func (f *feature) launchedProgramShouldExitWithZeroStatus() error {
	if err := f.waitLaunched(); err != nil {
		return err
	}
	return f.lastErr
}

func (f *feature) waitLaunched() error {
	defer f.console.Close()
	c := make(chan struct{})
	go func() {
		defer close(c)
		f.wg.Wait()
	}()

	select {
	case <-c:
		return nil
	case <-time.After(20 * time.Second):
		return fmt.Errorf("test %s timed out", f.scenarioID)
	}
}

func (f *feature) launchedProgramShouldExitWithNonZeroStatus() error {
	if err := f.waitLaunched(); err != nil {
		return err
	}
	if f.lastErr == nil {
		return errors.New("program returned zero exit code")
	}
	return nil
}

func (f *feature) tagValue(stack *cloudformation.Stack, tagKey string) string {
	for _, t := range stack.Tags {
		if aws.StringValue(t.Key) == tagKey {
			return aws.StringValue(t.Value)
		}
	}
	return ""
}

func (f *feature) replaceParameters(s string) string {
	s = strings.Replace(s, "%scenarioid%", f.scenarioID, -1)
	s = strings.Replace(s, "%featureid%", f.featureID, -1)
	s = strings.Replace(s, "%aws_profile%", os.Getenv("AWS_PROFILE"), -1)
	s = strings.Replace(s, "%testdir%", f.testDir, -1)

	return s
}

func (f *feature) fileShouldContainExactly(fname string, content *gherkin.DocString) error {
	c := f.replaceParameters(content.Content)

	file, err := f.fs.Open(fname)
	if err != nil {
		return err
	}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(buf)) != strings.TrimSpace(c) {
		return fmt.Errorf("file content is not equal to the expected string. File contents:\n%s", string(buf))
	}
	return nil
}

func FeatureContext(s *godog.Suite) {
	f := feature{}

	cfg := saaws.Config{}
	cfg.Profile = os.Getenv("AWS_PROFILE")

	s.Step(`^file "([^"]*)" exists:$`, f.fileExists)
	s.Step(`^I successfully run "([^"]*)"$`, f.iSuccessfullyRun)
	s.Step(`^stack "([^"]*)" should have status "([^"]*)"$`, f.stackShouldHaveStatus)
	s.Step(`^stack "([^"]*)" should not exist$`, f.stackShouldNotExist)
	s.Step(`^I modify file "([^"]*)":$`, f.iModifyFile)
	s.Step(`^I run "([^"]*)"$`, f.iRun)
	s.Step(`^exit code should not be zero$`, f.exitCodeShouldNotBeZero)
	s.Step(`^output should contain:$`, f.outputShouldContain)
	s.Step(`^output should be exactly:$`, f.outputShouldBeExactly)
	s.Step(`^node "([^"]*)" in json output should be:$`, f.nodeInJsonOutputShouldBe)
	s.Step(`^there should be stack "([^"]*)" that matches:$`, f.thereShouldBeStackThatMatches)
	s.Step(`^I launched "([^"]*)"$`, f.iLaunched)
	s.Step(`^terminal shows:$`, f.terminalShows)
	s.Step(`^I enter "([^"]*)"$`, f.iEnter)
	s.Step(`^launched program should exit with zero status$`, f.launchedProgramShouldExitWithZeroStatus)
	s.Step(`^launched program should exit with non zero status$`, f.launchedProgramShouldExitWithNonZeroStatus)
	s.Step(`^file "([^"]*)" should contain exactly:$`, f.fileShouldContainExactly)

	re := regexp.MustCompile("\\W")

	s.BeforeScenario(func(gs interface{}) {
		scenario := gs.(*gherkin.Scenario)
		f.scenarioName = re.ReplaceAllString(scenario.Name, "-")
		f.scenarioID = fmt.Sprintf("%.80s-%d", f.scenarioName, rand.Int63())

		f.cf = mock.New(f.scenarioName, f.featureID, f.scenarioID).Must(cfg).CF
		f.testDir = filepath.Join(".tmp", "stas_test_"+f.scenarioID)
		f.fs = vfs{
			afero.NewBasePathFs(afero.NewOsFs(), f.testDir),
		}
	})
	s.AfterScenario(func(interface{}, error) {
		f.fs.RemoveAll(".")
	})

	s.BeforeFeature(func(*gherkin.Feature) {
		f.featureID = strconv.FormatInt(rand.Int63(), 10)
	})

	s.AfterFeature(func(*gherkin.Feature) {
		if mock.IsMockEnabled() {
			return
		}

		stacks, err := f.cf.DescribeStacks(&cloudformation.DescribeStacksInput{})
		if err != nil {
			panic(err)
		}

		for _, s := range stacks.Stacks {
			if f.tagValue(s, "STAS_TEST") == f.featureID {
				f.cf.DeleteStack(&cloudformation.DeleteStackInput{
					StackName: s.StackName,
				})
			}
		}
	})
}

type assertionResult struct {
	err error
}

func (a *assertionResult) Errorf(format string, args ...interface{}) {
	a.err = fmt.Errorf(format, args...)
}

type vfs struct {
	afero.Fs
}

func (fs vfs) Open(name string) (conf.ReadSeekCloser, error) { return fs.Fs.Open(name) }
