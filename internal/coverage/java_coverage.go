package coverage

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

type jacocoReport struct {
	XMLName  xml.Name        `xml:"report"`
	Packages []jacocoPackage `xml:"package"`
	Counters []jacocoCounter `xml:"counter"`
}

type jacocoPackage struct {
	Name       string             `xml:"name,attr"`
	SourceFile []jacocoSourceFile `xml:"sourcefile"`
}

type jacocoSourceFile struct {
	Name     string          `xml:"name,attr"`
	Lines    []jacocoLine    `xml:"line"`
	Counters []jacocoCounter `xml:"counter"`
}

type jacocoLine struct {
	Number  string `xml:"nr,attr"`
	Missed  string `xml:"mi,attr"`
	Covered string `xml:"ci,attr"`
}

type jacocoCounter struct {
	Type    string `xml:"type,attr"`
	Missed  int    `xml:"missed,attr"`
	Covered int    `xml:"covered,attr"`
}

// ParseJaCoCoCoverage parses JaCoCo XML reports.
func ParseJaCoCoCoverage(profileData string) (*types.CoverageReport, error) {
	content, err := coverageInputContent(profileData)
	if err != nil {
		return nil, err
	}

	var reportXML jacocoReport
	if err := xml.Unmarshal([]byte(content), &reportXML); err != nil {
		return nil, fmt.Errorf("JaCoCo 覆盖率数据不是有效的 XML: %w", err)
	}
	if len(reportXML.Packages) == 0 {
		return nil, fmt.Errorf("JaCoCo XML 中没有 package 数据")
	}

	var files []types.CoverageFile
	var uncoveredFiles []string
	totalLines, coveredLines := jacocoLineTotals(reportXML.Counters)

	for _, pkg := range reportXML.Packages {
		for _, source := range pkg.SourceFile {
			cf := types.CoverageFile{Path: jacocoFilePath(pkg.Name, source.Name)}
			fileTotal, fileCovered := jacocoLineTotals(source.Counters)
			for _, line := range source.Lines {
				block, ok := jacocoCoverageBlock(line)
				if ok {
					cf.Blocks = append(cf.Blocks, block)
				}
			}
			if fileTotal == 0 {
				fileTotal, fileCovered = coverageFileLineStats(cf.Blocks)
			}
			cf.Percent = percent(fileCovered, fileTotal)
			if cf.Percent == 0 {
				uncoveredFiles = append(uncoveredFiles, cf.Path)
			}
			files = append(files, cf)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("JaCoCo XML 中没有 sourcefile 数据")
	}
	if totalLines == 0 {
		for _, file := range files {
			fileTotal, fileCovered := coverageFileLineStats(file.Blocks)
			totalLines += fileTotal
			coveredLines += fileCovered
		}
	}

	report := &types.CoverageReport{
		Framework:    "junit",
		TotalPercent: percent(coveredLines, totalLines),
		Files:        files,
		Summary: types.CoverageSummary{
			TotalStatements:   totalLines,
			CoveredStatements: coveredLines,
			TotalFiles:        len(files),
			CoveredFiles:      len(files) - len(uncoveredFiles),
			UncoveredFiles:    uncoveredFiles,
		},
	}
	report.Suggestions = GenerateSuggestions(report)
	report.TestTasks = GenerateTestTasks(report)
	return report, nil
}

func jacocoLineTotals(counters []jacocoCounter) (total, covered int) {
	for _, counter := range counters {
		if counter.Type != "LINE" {
			continue
		}
		return counter.Missed + counter.Covered, counter.Covered
	}
	return 0, 0
}

func jacocoCoverageBlock(line jacocoLine) (types.CoverageBlock, bool) {
	lineNumber, err := strconv.Atoi(strings.TrimSpace(line.Number))
	if err != nil {
		return types.CoverageBlock{}, false
	}
	covered, _ := strconv.Atoi(strings.TrimSpace(line.Covered))
	count := covered
	return types.CoverageBlock{
		StartLine: lineNumber,
		EndLine:   lineNumber,
		Count:     count,
		Covered:   covered > 0,
	}, true
}

func jacocoFilePath(packageName, sourceName string) string {
	if packageName == "" {
		return sourceName
	}
	return filepath.ToSlash(filepath.Join(strings.ReplaceAll(packageName, ".", "/"), sourceName))
}
