package report

import (
	// #nosec
	"crypto/md5"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/ozontech/allure-go/pkg/allure"
)

type (
	DefaultReport struct {
		mandatoryLabels []string
		specReport      types.SpecReport
		labelScraper    LabelScraper
	}
	LabelScraper interface {
		CheckMandatoryLabels([]string) error
		CreateAllureLabels() []*allure.Label
		GetID(uuid.UUID) (uuid.UUID, error)
		GetDescription(string) string
		GetAttachmentsFromReportEntries([]types.ReportEntry) map[string]string
	}
	Opt func(o *DefaultReport)
)

func WithMandatoryLabels(labels []string) Opt {
	return func(o *DefaultReport) {
		o.mandatoryLabels = labels
	}
}

func NewReport(specReport types.SpecReport, opts ...Opt) *DefaultReport {
	r := &DefaultReport{
		mandatoryLabels: []string{},
		specReport:      specReport,
	}
	ls := NewLabelScraper(specReport.LeafNodeText, specReport.LeafNodeLabels)
	r.SetLabelsScraper(ls)
	for _, o := range opts {
		o(r)
	}
	return r
}

func (r *DefaultReport) SetLabelsScraper(ls LabelScraper) {
	r.labelScraper = ls
}

func (r *DefaultReport) GenerateAllureReport(steps []*allure.Step) (allure.Result, error) {
	emptyReport := allure.Result{}
	id, err := r.labelScraper.GetID(uuid.New())
	if err != nil {
		return emptyReport, err
	}
	err = r.labelScraper.CheckMandatoryLabels(r.mandatoryLabels)
	if err != nil {
		return emptyReport, err
	}
	testCaseID := GetMD5Hash(id.String())

	defaultDescription := strings.Join(append(r.specReport.ContainerHierarchyTexts,
		r.specReport.LeafNodeText), " ")
	description := r.labelScraper.GetDescription(defaultDescription)

	reportStatus := allure.Passed
	statusDetails := allure.StatusDetail{}
	if r.specReport.Failure.TimelineLocation.Order != 0 {
		reportStatus = allure.Failed
		statusDetails.Message = r.specReport.Failure.Message
		statusDetails.Trace = r.specReport.Failure.Location.FullStackTrace
	}

	var allureAttachments []*allure.Attachment
	attachmentsMap := r.labelScraper.GetAttachmentsFromReportEntries(r.specReport.ReportEntries)

	for attachmentName, fileContent := range attachmentsMap {
		if fileContent == "" {
			continue
		}
		var contentType allure.MimeType = "application/octet-stream" // Default content type

		text := filepath.Ext(attachmentName) // Use attachmentName for extension as it's the intended file name

		switch strings.ToLower(text) {
		case ".json":
			contentType = "application/json"
		case ".xml":
			contentType = "application/xml"
		case ".html", ".htm":
			contentType = "text/html"
		case ".png":
			contentType = "image/png"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".gif":
			contentType = "image/gif"
		case ".svg":
			contentType = "image/svg+xml"
		case ".pdf":
			contentType = "application/pdf"
		case ".zip":
			contentType = "application/zip"
		case ".txt", ".log":
			contentType = "text/plain"
		default:
			contentType = "application/octet-stream"
		}

		allureAttachments = append(allureAttachments, allure.NewAttachment(attachmentName, contentType, []byte(fileContent)))
	}

	return allure.Result{
		Name:          r.specReport.LeafNodeText,
		Description:   description,
		FullName:      id.String(),
		StatusDetails: statusDetails,
		Status:        reportStatus,
		Start:         r.specReport.StartTime.UnixMilli(),
		Stop:          r.specReport.EndTime.UnixMilli(),
		Steps:         steps,
		UUID:          id,
		TestCaseID:    testCaseID,
		HistoryID:     GetMD5Hash(testCaseID),
		Labels:        r.labelScraper.CreateAllureLabels(),
		Attachments:   allureAttachments,
		ToPrint:       true,
	}, nil
}

func GetMD5Hash(text string) string {
	// #nosec
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
