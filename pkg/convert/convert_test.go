package convert_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Moon1706/ginkgo2allure/pkg/convert"
	"github.com/Moon1706/ginkgo2allure/pkg/convert/parser"
	"github.com/Moon1706/ginkgo2allure/pkg/convert/report"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/stretchr/testify/assert"
)

var (
	errTest = errors.New("test error")
)

type (
	mockReport    struct{}
	mockTransform struct {
		Err error
	}
	mockFileManager struct {
		SaveErr          error
		SavedResult      allure.Result
		SavedAttachments []*allure.Attachment
	}
)

func (m mockReport) GenerateAllureReport(_ []*allure.Step) (allure.Result, error) {
	return allure.Result{}, nil
}
func (m mockReport) SetLabelsScraper(_ report.LabelScraper) {}

func (m mockTransform) AnalyzeEvents(_ types.SpecEvents, _ types.Failure) error {
	return m.Err
}
func (m mockTransform) GetAllureSteps() []*allure.Step {
	return []*allure.Step{}
}
func (m *mockFileManager) SaveJSONResult(result allure.Result) error {
	m.SavedResult = result
	return m.SaveErr
}
func (m *mockFileManager) SaveAttachments(result allure.Result) error {
	m.SavedAttachments = result.Attachments
	return m.SaveErr
}

func TestConvertGinkgoToAllureReport(t *testing.T) {
	ginkgoReports := []types.Report{{
		SuiteDescription: "test",
		SpecReports: types.SpecReports{types.SpecReport{
			LeafNodeType: types.NodeTypeBeforeAll,
		}, types.SpecReport{
			LeafNodeType: types.NodeTypeIt,
		}},
	}}
	var tests = []struct {
		name       string
		createFunc parser.CreationFunc
		results    []allure.Result
		err        error
	}{{
		name: "correct",
		createFunc: func(specReport types.SpecReport, _ parser.Config) (*parser.Parser, error) {
			return parser.NewParser(specReport, mockTransform{Err: nil}, nil, mockReport{}), nil
		},
		results: []allure.Result{{}},
		err:     nil,
	}, {
		name: "error in parser creation function",
		createFunc: func(specReport types.SpecReport, _ parser.Config) (*parser.Parser, error) {
			return &parser.Parser{}, errTest
		},
		results: []allure.Result{},
		err:     errTest,
	}, {
		name: "error in parser",
		createFunc: func(specReport types.SpecReport, _ parser.Config) (*parser.Parser, error) {
			return parser.NewParser(specReport, mockTransform{Err: errTest}, nil, mockReport{}), nil
		},
		results: []allure.Result{},
		err:     errTest,
	}}

	for _, tt := range tests {
		results, err := convert.GinkgoToAllureReport(ginkgoReports, tt.createFunc, parser.Config{})
		assert.Equal(t, tt.err, err, fmt.Sprintf("got expected error (%s)", tt.name))
		assert.Equal(t, tt.results, results, fmt.Sprintf("got expected results (%s)", tt.name))
	}
}

func TestConvertPrintAllureReports(t *testing.T) {
	stringAttachment := allure.Result{
		Name: "Test with string attachment",
		Attachments: []*allure.Attachment{
			allure.NewAttachment("test_file.txt", "text/plain", []byte("file content")),
		},
	}
	binaryAttachment := allure.Result{
		Name: "Test with binary attachment",
		Attachments: []*allure.Attachment{
			allure.NewAttachment("foo.tar.gz", "application/octet-stream", []byte{0, 1, 2, 3, 4, 5}),
		},
	}

	var tests = []struct {
		name            string
		results         []allure.Result
		expectedResult  allure.Result
		mockFileManager *mockFileManager
		errs            []error
	}{
		{
			name:           "correct",
			results:        []allure.Result{{}},
			expectedResult: stringAttachment,
			mockFileManager: &mockFileManager{
				SaveErr: nil,
			},
			errs: []error{},
		},
		{
			name:           "wrong",
			results:        []allure.Result{{}},
			expectedResult: stringAttachment,
			mockFileManager: &mockFileManager{
				SaveErr: errTest,
			},
			errs: []error{errTest, errTest},
		},
		{
			name:           "with string attachment",
			results:        []allure.Result{stringAttachment},
			expectedResult: stringAttachment,
			mockFileManager: &mockFileManager{
				SaveErr: nil,
			},
			errs: []error{},
		},
		{
			name:           "with binary attachment",
			results:        []allure.Result{stringAttachment},
			expectedResult: binaryAttachment,
			mockFileManager: &mockFileManager{
				SaveErr: nil,
			},
			errs: []error{},
		},
	}

	for _, tt := range tests {
		tt.mockFileManager.SavedResult = allure.Result{} // Reset for each test run
		tt.mockFileManager.SavedAttachments = nil        // Reset for each test run

		errs := convert.PrintAllureReports(tt.results, tt.mockFileManager)
		assert.Equal(t, tt.errs, errs, fmt.Sprintf("got expected errors (%s)", tt.name))

		if len(tt.results) > 0 && tt.mockFileManager.SaveErr == nil {
			assert.Equal(t, tt.results[0], tt.mockFileManager.SavedResult, fmt.Sprintf("saved result mismatch (%s)", tt.name))

			if tt.name == "with attachment" {
				assert.NotEmpty(t, tt.mockFileManager.SavedAttachments, "should have attachments")
				assert.Len(t, tt.mockFileManager.SavedAttachments, 1, "should have one attachment")
				actualAttachment := tt.mockFileManager.SavedAttachments[0]
				expectedAttachment := tt.expectedResult.Attachments[0]
				assert.Equal(t, expectedAttachment.Name, actualAttachment.Name, "attachment name should match")
				assert.Equal(t, expectedAttachment.Type, actualAttachment.Type, "attachment type should match")
				assert.Equal(t, expectedAttachment.GetContent(), actualAttachment.GetContent(), "attachment content should match")
			}
		}
	}
}
