package awscf

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type chSetParams struct {
	builtPP     []*cloudformation.Parameter
	missingKeys []string

	providedPP map[string]string

	stack *Stack
	err   error
}

type ParametersMissingError struct {
	MissingParameters []string
}

func (e *ParametersMissingError) Error() string {
	return fmt.Sprintf(
		"the following parameters are required but not provided: %s",
		strings.Join(e.MissingParameters, ", "),
	)
}

func newChSetParams(stack *Stack, providedPP map[string]string) *chSetParams {
	return &chSetParams{stack: stack, providedPP: providedPP}
}

func (cp *chSetParams) collect() ([]*cloudformation.Parameter, error) {
	if cp.err != nil {
		return cp.builtPP, cp.err
	}

	if len(cp.missingKeys) > 0 {
		err := &ParametersMissingError{}
		err.MissingParameters = cp.missingKeys

		return cp.builtPP, err
	}

	return cp.builtPP, nil
}

func (cp *chSetParams) add(bodyParam *cloudformation.TemplateParameter) {
	if cp.err != nil {
		return
	}

	paramKey := aws.StringValue(bodyParam.ParameterKey)

	if v, ok := cp.providedPP[paramKey]; ok {
		cp.builtPP = append(cp.builtPP, &cloudformation.Parameter{
			ParameterKey:   aws.String(paramKey),
			ParameterValue: aws.String(v),
		})

		return
	}

	if aws.StringValue(bodyParam.DefaultValue) != "" {
		return
	}

	deployed, err := cp.stack.AlreadyDeployed()
	if err != nil {
		cp.err = err
		return
	}

	if !deployed {
		cp.missingKeys = append(cp.missingKeys, paramKey)
		return
	}

	info, err := cp.stack.Info()
	if err != nil {
		cp.err = err
		return
	}

	if !info.HasParameter(paramKey) {
		cp.missingKeys = append(cp.missingKeys, paramKey)
		return
	}

	cp.builtPP = append(cp.builtPP, &cloudformation.Parameter{
		ParameterKey:     aws.String(paramKey),
		UsePreviousValue: aws.Bool(true),
	})
}
