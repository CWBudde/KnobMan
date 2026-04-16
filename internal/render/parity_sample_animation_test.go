package render

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestParityRegressionAnimatedSamplesKeyframes(t *testing.T) {
	root := testRepoRoot(t)
	runNamedAnimatedParitySuite(t, root, animatedSampleParitySuite(root), animatedSampleNames(), "baseline-go")
}

func TestAnimatedSampleKeyframeCheckpoints(t *testing.T) {
	root := testRepoRoot(t)
	summary := collectNamedAnimatedKeyframeCheckpointSummary(t, root, animatedSampleParitySuite(root), animatedSampleNames(), "baseline-java")

	diffRate := 0.0
	if summary.TotalPixels > 0 {
		diffRate = float64(summary.TotalDiffPixels) / float64(summary.TotalPixels)
	}

	t.Logf(
		"animated sample checkpoint baseline=%s compared=%d diffCases=%d meanRMSE=%.4f maxRMSE=%.4f diffRate=%.4f worst=%s",
		"baseline-java",
		summary.Compared,
		summary.DiffCases,
		summary.MeanRMSE,
		summary.MaxRMSE,
		diffRate,
		summary.WorstCase,
	)

	budget := parityCheckpointBudget{
		ComparedCases:    9,
		ComparedMaxRMSE:  226,
		ComparedMeanRMSE: 140,
		ComparedDiffRate: 0.96,
	}

	if summary.Compared != budget.ComparedCases {
		t.Fatalf("compared cases = %d, want %d", summary.Compared, budget.ComparedCases)
	}

	if summary.MaxRMSE > budget.ComparedMaxRMSE {
		t.Fatalf("max RMSE %.4f exceeded checkpoint %.4f", summary.MaxRMSE, budget.ComparedMaxRMSE)
	}

	if summary.MeanRMSE > budget.ComparedMeanRMSE {
		t.Fatalf("mean RMSE %.4f exceeded checkpoint %.4f", summary.MeanRMSE, budget.ComparedMeanRMSE)
	}

	if diffRate > budget.ComparedDiffRate {
		t.Fatalf("diff rate %.4f exceeded checkpoint %.4f", diffRate, budget.ComparedDiffRate)
	}
}

func animatedSampleParitySuite(root string) paritySuite {
	return paritySuite{
		name:            "animated-samples",
		sampleDir:       filepath.Join(root, "assets", "samples"),
		baselineGoDir:   filepath.Join(root, "tests", "parity", "animated-samples", "baseline-go"),
		baselineJavaDir: filepath.Join(root, "tests", "parity", "animated-samples", "baseline-java"),
		artifactsDir:    filepath.Join(root, "tests", "parity", "animated-samples", "artifacts"),
	}
}

func animatedSampleNames() []string {
	return []string{
		"Green_Radar",
		"LineShadow",
		"White_Vol",
	}
}

func collectNamedAnimatedKeyframeCheckpointSummary(t *testing.T, root string, suite paritySuite, names []string, baseline string) parityCheckpointSummary {
	t.Helper()

	refDir := suite.baselineDir(t, baseline)
	summary := parityCheckpointSummary{}

	for _, sample := range names {
		samplePath := filepath.Join(suite.sampleDir, sample+".knob")

		doc, textures, err := LoadParityDocument(samplePath, root)
		if err != nil {
			t.Fatalf("load sample %s: %v", samplePath, err)
		}

		for _, keyframe := range animatedKeyframes() {
			frame := keyframe.frameIndex(doc.Prefs.RenderFrames.Val)
			refPath := filepath.Join(refDir, sample+"__"+keyframe.name+".png")

			ref, err := ReadPNGAsRGBA(refPath)
			if err != nil {
				t.Fatalf("read reference %s: %v", refPath, err)
			}

			out := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
			RenderFrame(out, doc, frame, textures)

			rmse, diffPixels, totalPixels, diffRatio := comparePixBufMetrics(out, ref)
			summary.Compared++
			summary.TotalPixels += totalPixels
			summary.TotalDiffPixels += diffPixels
			summary.MeanRMSE += rmse

			if diffPixels != 0 {
				summary.DiffCases++
			}

			if rmse > summary.MaxRMSE {
				summary.MaxRMSE = rmse
				summary.WorstCase = fmt.Sprintf("%s__%s (rmse=%.4f, diff=%.4f)", sample, keyframe.name, rmse, diffRatio)
			}
		}
	}

	if summary.Compared > 0 {
		summary.MeanRMSE /= float64(summary.Compared)
	}

	return summary
}
