// +build acceptance

package tests

import (
	"context"
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
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	"github.com/DATA-DOG/godog/gherkin"
	expect "github.com/Netflix/go-expect"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	yaml "gopkg.in/yaml.v2"
)

var opt = godog.Options{
	Paths:  []string{"."},
	Output: colors.Colored(os.Stdout),
	// Format:      "pretty",
	Format:      "progress",
	Concurrency: 5,
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

	console *expect.Console
	lastCmd *exec.Cmd
	cancel  context.CancelFunc

	cf *cloudformation.CloudFormation
}

func (f *feature) fileExists(fname string, content *gherkin.DocString) error {
	fpath := filepath.Join(f.testDir, fname)
	dir, _ := filepath.Split(fpath)

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	c := f.replaceParameters(content.Content)

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
		return fmt.Errorf("Stack %s is not supposed to exist", s)
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
		return fmt.Errorf("program returned zero exit code. Programs output: \n%s", f.lastOutput)
	}
	return nil
}

func (f *feature) outputShouldContain(s *gherkin.DocString) error {
	if !strings.Contains(f.lastOutput, f.replaceParameters(s.Content)) {
		return fmt.Errorf("output doesn't contain searched string. Output:\n%s", f.lastOutput)
	}
	return nil
}

func (f *feature) outputShouldBeExactly(s *gherkin.DocString) error {
	if strings.TrimSpace(f.lastOutput) != strings.TrimSpace(f.replaceParameters(s.Content)) {
		return fmt.Errorf("output isn't equal to expected string. Output:\n%s", f.lastOutput)
	}
	return nil
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

	bin, err := filepath.Abs("../bin/stas")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	f.cancel = cancel
	cmd := exec.CommandContext(ctx, bin, strings.Split(cmdInstruction, " ")...)
	cmd.Dir = f.testDir

	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	err = cmd.Start()
	if err != nil {
		return err
	}

	f.lastCmd = cmd
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
	defer f.console.Close()
	defer f.cancel()
	return f.lastCmd.Wait()
}

func (f *feature) launchedProgramShouldExitWithNonZeroStatus() error {
	defer f.console.Close()
	defer f.cancel()
	err := f.lastCmd.Wait()
	if err == nil {
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
	s = strings.Replace(s, "%featureid%", f.featurID, -1)

	return s
}

func (f *feature) fileShouldContainExactly(fname string, content *gherkin.DocString) error {
	fpath := filepath.Join(f.testDir, fname)
	c := f.replaceParameters(content.Content)

	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(buf)) != strings.TrimSpace(c) {
		return fmt.Errorf("file content is not equal to the expected string. File contents:\n%s", string(buf))
	}
	return nil
}

func FeatureContext(s *godog.Suite) {
	testDir := "./.tmp"
	f := feature{}

	sess := session.Must(session.NewSession())
	cf := cloudformation.New(sess, (&aws.Config{}).WithMaxRetries(8))

	f.cf = cf

	s.Step(`^file "([^"]*)" exists:$`, f.fileExists)
	s.Step(`^I successfully run "([^"]*)"$`, f.iSuccessfullyRun)
	s.Step(`^stack "([^"]*)" should have status "([^"]*)"$`, f.stackShouldHaveStatus)
	s.Step(`^stack "([^"]*)" should not exist$`, f.stackShouldNotExist)
	s.Step(`^I modify file "([^"]*)":$`, f.iModifyFile)
	s.Step(`^I run "([^"]*)"$`, f.iRun)
	s.Step(`^exit code should not be zero$`, f.exitCodeShouldNotBeZero)
	s.Step(`^output should contain:$`, f.outputShouldContain)
	s.Step(`^output should be exactly:$`, f.outputShouldBeExactly)
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
		prefix := re.ReplaceAllString(scenario.Name, "-")
		f.scenarioID = fmt.Sprintf("%.80s-%d", prefix, rand.Int63())
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
			if f.tagValue(s, "STAS_TEST") == f.featurID {
				cf.DeleteStack(&cloudformation.DeleteStackInput{
					StackName: s.StackName,
				})
			}
		}
	})
}
