// +build acceptance

package tests

import (
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
)

var opt = godog.Options{
	Paths:  []string{"."},
	Output: colors.Colored(os.Stdout),
	Format: "pretty",
	// Format:      "progress",
	// Concurrency: 4,
	Randomize: time.Now().UTC().UnixNano(),
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)
	rand.Seed(time.Now().UnixNano())
}

func TestMain(m *testing.M) {
	flag.Parse()
	// opt.Paths = flag.Args()

	status := godog.RunWithOptions("stas", func(s *godog.Suite) {
		FeatureContext(s)
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}

	os.Exit(status)
}

type feature struct {
	testID  string
	testDir string

	cf *cloudformation.CloudFormation
}

func (f *feature) fileExists(fname string, content *gherkin.DocString) error {
	fpath := filepath.Join(f.testDir, fname)
	dir, _ := filepath.Split(fpath)

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	c := strings.Replace(content.Content, "%testid%", f.testID, -1)
	return ioutil.WriteFile(fpath, []byte(c), 0700)
}

func (f *feature) iSuccessfullyRun(cmd string) error {
	bin, err := filepath.Abs("../bin/stas")
	if err != nil {
		return err
	}

	c := exec.Command(bin, strings.Split(cmd, " ")...)
	c.Dir = f.testDir

	out, err := c.CombinedOutput()

	if err != nil {
		return fmt.Errorf("err: %v, output:\n%s", err, string(out))
	}

	return nil
}

func (f *feature) stackShouldHaveStatus(stackName, status string) error {
	s := strings.Replace(stackName, "%testid%", f.testID, -1)
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

func FeatureContext(s *godog.Suite) {
	testDir := "./.tmp"
	f := feature{}

	sess := session.Must(session.NewSession())
	cf := cloudformation.New(sess)

	f.cf = cf

	s.Step(`^file "([^"]*)" exists:$`, f.fileExists)
	s.Step(`^I successfully run "([^"]*)"$`, f.iSuccessfullyRun)
	s.Step(`^stack "([^"]*)" should have status "([^"]*)"$`, f.stackShouldHaveStatus)

	s.BeforeScenario(func(interface{}) {
		f.testID = strconv.FormatInt(rand.Int63(), 10)
		f.testDir = filepath.Join(testDir, "stas_test_"+f.testID)
	})

	s.AfterSuite(func() {
		os.RemoveAll(testDir)

		stacks, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{})
		if err != nil {
			panic(err)
		}

		for _, s := range stacks.Stacks {
			for _, t := range s.Tags {
				if aws.StringValue(t.Key) == "STAS_TEST" {
					cf.DeleteStack(&cloudformation.DeleteStackInput{
						StackName: s.StackName,
					})
				}
			}
		}
	})
}
