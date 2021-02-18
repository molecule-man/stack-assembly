package mock

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

var dumpersPool = map[string]*dumper{}
var mu = sync.Mutex{}

func newDumper(testID, featureID, scenarioID string) *dumper {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := dumpersPool[scenarioID]; !ok {
		dumpersPool[scenarioID] = &dumper{
			rr:          registries{},
			timesDumped: map[string]uint{},
			testID:      testID,
			featureID:   featureID,
			scenarioID:  scenarioID,
			dir:         "goldenfiles",
		}
		dumpersPool[scenarioID].addReplacement(featureID, "FEATURE_ID")
	}

	return dumpersPool[scenarioID]
}

type dumper struct {
	sync.Mutex

	rr          registries
	timesDumped map[string]uint
	testID      string
	featureID   string
	scenarioID  string

	dir string
}

func (d *dumper) dump(methodName string, input interface{}, output interface{}, err error) {
	d.Lock()
	defer d.Unlock()

	var dumpedErr *dumpedError
	if err != nil {
		dumpedErr = &dumpedError{
			Err: err.Error(),
		}

		var aerr awserr.Error
		if errors.As(err, &aerr) {
			dumpedErr.Code = aerr.Code()
			dumpedErr.Msg = aerr.Message()
		}
	}

	data := map[string]interface{}{
		"input":  input,
		"output": output,
		"err":    dumpedErr,
	}

	d.dumpFile(d.fname(methodName, input), data)
}

func (d *dumper) read(methodName string, input interface{}, output interface{}) error {
	d.Lock()
	defer d.Unlock()

	data := struct {
		Output interface{}
		Err    *dumpedError
	}{Output: output}

	buf, err := ioutil.ReadFile(d.fname(methodName, input))
	if err != nil {
		jsonF, jErr := json.MarshalIndent(input, "", "  ")
		if jErr != nil {
			log.Fatal(jErr) //nolint:gocritic
		}

		fmt.Printf("json = %+v\n", string(jsonF))

		jsonFStr := d.rr.replace(d.scenarioID, string(jsonF))
		fmt.Printf("jsonStr = %+v\n", jsonFStr)

		panic(err)
	}

	content := d.rr.unReplace(d.scenarioID, string(buf))

	err = json.Unmarshal([]byte(content), &data)
	if err != nil {
		log.Fatal(err)
	}

	if data.Err != nil {
		if data.Err.Code != "" {
			return &awsError{data.Err}
		}

		return errors.New(data.Err.Err)
	}

	return nil
}

func (d *dumper) fname(methodName string, input interface{}) string {
	fname := d.testID + "-" + methodName + "-" + d.hash(input)

	if _, ok := d.timesDumped[fname]; !ok {
		d.timesDumped[fname] = 0
	}

	d.timesDumped[fname]++

	return fmt.Sprintf("%s/%s-%d.json", d.dir, fname, d.timesDumped[fname])
}

func (d *dumper) dumpFile(f string, data interface{}) {
	json, err := json.MarshalIndent(data, "", "  ")

	if err != nil {
		log.Fatal(err)
	}

	jsonStr := d.rr.replace(d.scenarioID, string(json))

	err = ioutil.WriteFile(f, []byte(jsonStr), 0644)

	if err != nil {
		log.Fatal(err)
	}
}
func (d *dumper) hash(input interface{}) string {
	js, err := json.Marshal(input)
	if err != nil {
		log.Fatal(err)
	}

	jsonStr := d.rr.replace(d.scenarioID, string(js))
	buf := md5.Sum([]byte(jsonStr))

	return hex.EncodeToString(buf[:])
}

func (d *dumper) addReplacement(from, to string) {
	r, ok := d.rr[d.scenarioID]

	if !ok {
		r = replacementRegistry{
			rMap:         map[string]string{},
			rRevMap:      map[string]string{},
			usedMasks:    map[string]uint{},
			replacements: []string{},
		}

		r.add(d.scenarioID, "SCENARIO_ID")
	}

	r.add(from, to)

	d.rr[d.scenarioID] = r
}

type registries map[string]replacementRegistry

func (rr registries) replace(id, content string) string {
	r := rr[id]
	return r.replace(content)
}
func (rr registries) unReplace(id, content string) string {
	r := rr[id]
	return r.unReplace(content)
}

type replacementRegistry struct {
	rMap         map[string]string
	rRevMap      map[string]string
	usedMasks    map[string]uint
	replacements []string
}

func (rr *replacementRegistry) add(from, to string) {
	if _, ok := rr.rMap[from]; ok {
		return
	}

	if timesUsed, ok := rr.usedMasks[to]; ok {
		rr.usedMasks[to] = timesUsed + 1
		to = fmt.Sprintf("%s-%d", to, timesUsed+1)
	} else {
		rr.usedMasks[to] = 1
	}

	rr.rMap[from] = "%" + to + "%"
	rr.rRevMap["%"+to+"%"] = from

	rr.replacements = append(rr.replacements, from)

	sort.Slice(rr.replacements, func(i, j int) bool {
		return len(rr.replacements[i]) > len(rr.replacements[j])
	})
}

func (rr replacementRegistry) replace(content string) string {
	for _, from := range rr.replacements {
		to := rr.rMap[from]
		content = strings.ReplaceAll(content, from, to)
	}

	return content
}

func (rr replacementRegistry) unReplace(content string) string {
	for from, to := range rr.rRevMap {
		content = strings.ReplaceAll(content, from, to)
	}

	return content
}

type dumpedError struct {
	Err  string
	Code string
	Msg  string
}

type awsError struct {
	de *dumpedError
}

func (e awsError) Error() string {
	return e.de.Err
}

func (e awsError) Code() string {
	return e.de.Code
}

func (e awsError) Message() string {
	return e.de.Code
}

func (e awsError) OrigErr() error {
	return nil
}
