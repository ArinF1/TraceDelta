package report

import (
	"bytes"
	"fmt"
	"html/template"
	"io"

	"github.com/ArinF1/TraceDelta/internal/diff"
)

type htmlReport struct {
	Title         string
	OutcomeClass  string
	OutcomeLabel  string
	OutcomeDetail string
	BaselinePath  string
	CandidatePath string
	AddedSpans    int
	RemovedSpans  int
	ChangedSpans  int
	FindingCount  int
	Findings      []htmlFinding
}

type htmlFinding struct {
	Kind        string
	KindClass   string
	SpanName    string
	ServiceName string
	Occurrence  int
	Evidence    string
}

// WriteHTML writes one self-contained offline report. All dynamic content is
// escaped by html/template and the complete document is buffered before write.
func WriteHTML(w io.Writer, result diff.Result, metadata Metadata) error {
	report := makeHTMLReport(result, metadata)
	tmpl, err := template.New("report").Parse(htmlReportTemplate)
	if err != nil {
		return fmt.Errorf("prepare HTML report: %w", err)
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, report); err != nil {
		return fmt.Errorf("encode HTML report: %w", err)
	}
	if _, err := w.Write(output.Bytes()); err != nil {
		return fmt.Errorf("write HTML report: %w", err)
	}
	return nil
}

func makeHTMLReport(result diff.Result, metadata Metadata) htmlReport {
	report := htmlReport{
		Title:         "TraceDelta report — No behavioral differences",
		OutcomeClass:  "pass",
		OutcomeLabel:  "No behavioral differences",
		OutcomeDetail: "The candidate passed the configured comparison policy.",
		BaselinePath:  escapeControls(metadata.BaselinePath),
		CandidatePath: escapeControls(metadata.CandidatePath),
		AddedSpans:    result.AddedSpans,
		RemovedSpans:  result.RemovedSpans,
		ChangedSpans:  result.ChangedSpans,
		FindingCount:  len(result.Changes),
		Findings:      make([]htmlFinding, len(result.Changes)),
	}
	if result.HasDifferences() {
		report.Title = "TraceDelta report — Behavioral differences"
		report.OutcomeClass = "differences"
		report.OutcomeLabel = "Behavioral differences detected"
		report.OutcomeDetail = "The candidate did not pass the configured comparison policy."
	}
	for index, change := range result.Changes {
		serviceName := escapeControls(change.ServiceName)
		if serviceName == "" {
			serviceName = "(unknown)"
		}
		report.Findings[index] = htmlFinding{
			Kind:        string(change.Kind),
			KindClass:   htmlKindClass(change.Kind),
			SpanName:    escapeControls(change.SpanName),
			ServiceName: serviceName,
			Occurrence:  change.Occurrence + 1,
			Evidence:    htmlEvidence(change),
		}
	}
	return report
}

func htmlKindClass(kind diff.Kind) string {
	switch kind {
	case diff.KindAdded:
		return "added"
	case diff.KindRemoved:
		return "removed"
	case diff.KindChanged:
		return "changed"
	default:
		return "unknown"
	}
}

func htmlEvidence(change diff.Change) string {
	switch change.Kind {
	case diff.KindAdded:
		return "Present only in candidate"
	case diff.KindRemoved:
		return "Present only in baseline"
	}
	switch change.Field {
	case diff.FieldStatus:
		return fmt.Sprintf("status %s → %s", escapeControls(change.Before), escapeControls(change.After))
	case diff.FieldErrorType:
		return fmt.Sprintf("error.type %s → %s", formatEvidence(change.Before, change.BeforePresent), formatEvidence(change.After, change.AfterPresent))
	case diff.FieldDuration:
		return fmt.Sprintf(
			"duration %s → %s (thresholds: ≥%s and ≥%s)",
			escapeControls(change.Before),
			escapeControls(change.After),
			formatPercentage(change.DurationThresholdRelative),
			change.DurationThresholdAbsolute,
		)
	default:
		return "Behavioral change"
	}
}

const htmlReportTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="color-scheme" content="light">
  <title>{{.Title}}</title>
  <style>
    :root { color-scheme: light; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #f4f7fb; color: #172033; }
    * { box-sizing: border-box; }
    body { margin: 0; min-width: 320px; background: #f4f7fb; }
    main { width: min(1120px, calc(100% - 32px)); margin: 40px auto; }
    header { margin-bottom: 24px; }
    .eyebrow { margin: 0 0 8px; color: #56647a; font-size: 12px; font-weight: 750; letter-spacing: .12em; text-transform: uppercase; }
    h1 { margin: 0; font-size: clamp(28px, 4vw, 44px); letter-spacing: -.035em; }
    .outcome { margin-top: 20px; padding: 18px 20px; border: 1px solid; border-radius: 14px; }
    .outcome.pass { background: #ecfdf3; border-color: #86d4a0; color: #135c2d; }
    .outcome.differences { background: #fff7e8; border-color: #e7b85d; color: #704510; }
    .outcome strong { display: block; font-size: 18px; }
    .outcome p { margin: 4px 0 0; }
    .inputs { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin: 20px 0 24px; }
    .input { min-width: 0; padding: 14px 16px; border: 1px solid #d9e0ea; border-radius: 12px; background: #fff; }
    .label { display: block; margin-bottom: 6px; color: #647188; font-size: 12px; font-weight: 700; text-transform: uppercase; }
    code { overflow-wrap: anywhere; color: #27344c; font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace; font-size: 13px; }
    .summary { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 28px; }
    .metric { padding: 18px; border: 1px solid #d9e0ea; border-radius: 12px; background: #fff; }
    .metric b { display: block; font-size: 28px; line-height: 1; }
    .metric span { display: block; margin-top: 8px; color: #647188; font-size: 13px; }
    section { overflow: hidden; border: 1px solid #d9e0ea; border-radius: 14px; background: #fff; box-shadow: 0 12px 32px rgba(31, 46, 75, .07); }
    h2 { margin: 0; padding: 20px; border-bottom: 1px solid #e5eaf1; font-size: 18px; }
    .empty { margin: 0; padding: 34px 20px; color: #59677d; text-align: center; }
    .table-wrap { overflow-x: auto; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 14px 16px; border-bottom: 1px solid #edf0f5; text-align: left; vertical-align: top; }
    th { color: #647188; font-size: 12px; letter-spacing: .04em; text-transform: uppercase; }
    tbody tr:last-child td { border-bottom: 0; }
    .badge { display: inline-block; min-width: 78px; padding: 5px 8px; border-radius: 999px; font-size: 11px; font-weight: 800; text-align: center; }
    .badge.added { background: #e8f5ff; color: #165a8d; }
    .badge.removed { background: #fff0f0; color: #9b2c2c; }
    .badge.changed { background: #fff4d6; color: #73510d; }
    .badge.unknown { background: #eef1f5; color: #465267; }
    .span-name { font-weight: 700; }
    .muted { margin-top: 4px; color: #6b778b; font-size: 12px; }
    .evidence { font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace; font-size: 13px; overflow-wrap: anywhere; }
    footer { margin-top: 18px; color: #6b778b; font-size: 12px; text-align: center; }
    @media (max-width: 760px) { main { margin: 20px auto; } .inputs { grid-template-columns: 1fr; } .summary { grid-template-columns: 1fr 1fr; } th:nth-child(3), td:nth-child(3) { display: none; } }
  </style>
</head>
<body>
  <main>
    <header>
      <p class="eyebrow">TraceDelta comparison report</p>
      <h1>Runtime behavior comparison</h1>
      <div class="outcome {{.OutcomeClass}}" role="status">
        <strong>{{.OutcomeLabel}}</strong>
        <p>{{.OutcomeDetail}}</p>
      </div>
    </header>
    <div class="inputs" aria-label="Compared inputs">
      <div class="input"><span class="label">Baseline</span><code>{{.BaselinePath}}</code></div>
      <div class="input"><span class="label">Candidate</span><code>{{.CandidatePath}}</code></div>
    </div>
    <div class="summary" aria-label="Comparison summary">
      <div class="metric"><b>{{.AddedSpans}}</b><span>Added spans</span></div>
      <div class="metric"><b>{{.RemovedSpans}}</b><span>Removed spans</span></div>
      <div class="metric"><b>{{.ChangedSpans}}</b><span>Changed spans</span></div>
      <div class="metric"><b>{{.FindingCount}}</b><span>Total findings</span></div>
    </div>
    <section aria-labelledby="findings-title">
      <h2 id="findings-title">Findings</h2>
      {{if .Findings}}
      <div class="table-wrap">
        <table>
          <thead><tr><th>Change</th><th>Span</th><th>Service</th><th>Evidence</th></tr></thead>
          <tbody>
          {{range .Findings}}<tr>
            <td><span class="badge {{.KindClass}}">{{.Kind}}</span></td>
            <td><div class="span-name">{{.SpanName}}</div><div class="muted">Occurrence {{.Occurrence}}</div></td>
            <td>{{.ServiceName}}</td>
            <td class="evidence">{{.Evidence}}</td>
          </tr>{{end}}
          </tbody>
        </table>
      </div>
      {{else}}<p class="empty">No findings under the configured policy.</p>{{end}}
    </section>
    <footer>Generated locally by TraceDelta · Self-contained offline report</footer>
  </main>
</body>
</html>
`
