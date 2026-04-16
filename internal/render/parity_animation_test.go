package render

import (
	"os"
	"path/filepath"
	"testing"
)

type animatedKeyframe struct {
	name string
}

func TestParityRegressionAnimatedFixturesKeyframes(t *testing.T) {
	root := testRepoRoot(t)
	runNamedAnimatedParitySuite(t, root, animatedParitySuite(root), animatedFixtureNamesGo(), "baseline-go")
}

func TestParityGoldenAnimatedFixturesKeyframes(t *testing.T) {
	root := testRepoRoot(t)
	runNamedAnimatedParitySuite(t, root, animatedParitySuite(root), animatedFixtureNamesJava(), "baseline-java")
}

func TestAnimatedEffectFixtureCheckpoints(t *testing.T) {
	root := testRepoRoot(t)
	summary := collectNamedAnimatedKeyframeCheckpointSummary(t, root, animatedParitySuite(root), phase3AnimatedFixtureNames(), "baseline-java")

	diffRate := 0.0
	if summary.TotalPixels > 0 {
		diffRate = float64(summary.TotalDiffPixels) / float64(summary.TotalPixels)
	}

	t.Logf(
		"animated effect fixture checkpoint baseline=%s compared=%d diffCases=%d meanRMSE=%.4f maxRMSE=%.4f diffRate=%.4f worst=%s",
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
		ComparedMaxRMSE:  132,
		ComparedMeanRMSE: 88,
		ComparedDiffRate: 0.91,
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

func animatedParitySuite(root string) paritySuite {
	return paritySuite{
		name:            "animated",
		sampleDir:       filepath.Join(root, "tests", "parity", "animated", "inputs"),
		baselineGoDir:   filepath.Join(root, "tests", "parity", "animated", "baseline-go"),
		baselineJavaDir: filepath.Join(root, "tests", "parity", "animated", "baseline-java"),
		artifactsDir:    filepath.Join(root, "tests", "parity", "animated", "artifacts"),
	}
}

func runNamedAnimatedParitySuite(t *testing.T, root string, suite paritySuite, names []string, baseline string) {
	t.Helper()

	refDir := suite.baselineDir(t, baseline)
	artifactsDir := filepath.Join(suite.artifactsDir, baseline)

	for _, sample := range names {
		samplePath := filepath.Join(suite.sampleDir, sample+".knob")

		t.Run(sample, func(t *testing.T) {
			doc, textures, err := LoadParityDocument(samplePath, root)
			if err != nil {
				t.Fatalf("load sample: %v", err)
			}

			for _, keyframe := range animatedKeyframes() {
				frame := keyframe.frameIndex(doc.Prefs.RenderFrames.Val)
				refPath := filepath.Join(refDir, sample+"__"+keyframe.name+".png")

				t.Run(keyframe.name, func(t *testing.T) {
					if _, err := os.Stat(refPath); err != nil {
						t.Skipf("missing reference: %s", refPath)
					}

					out := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
					RenderFrame(out, doc, frame, textures)

					if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
						t.Fatalf("mkdir artifacts: %v", err)
					}

					actualPath := filepath.Join(artifactsDir, sample+"__"+keyframe.name+".png")
					if err := WritePixBufPNG(actualPath, out); err != nil {
						t.Fatalf("write artifact: %v", err)
					}

					ref, err := ReadPNGAsRGBA(refPath)
					if err != nil {
						t.Fatalf("read reference: %v", err)
					}

					if err := comparePixBufWithRef(out, ref, parityTolerance); err != nil {
						t.Fatalf("frame %d parity mismatch: %v", frame, err)
					}
				})
			}
		})
	}
}

func animatedFixtureNamesGo() []string {
	return []string{
		"anim_primitive_image_strip",
		"anim_effect_offset_image",
		"anim_layer_animstep_strip",
		"anim_frame_mask_ranges",
		"anim_effect_transform_mask",
		"anim_effect_shadow_color",
		"anim_effect_combo_stack",
	}
}

func animatedFixtureNamesJava() []string {
	return []string{
		"anim_primitive_image_strip",
		"anim_layer_animstep_strip",
		"anim_frame_mask_ranges",
	}
}

func phase3AnimatedFixtureNames() []string {
	return []string{
		"anim_effect_transform_mask",
		"anim_effect_shadow_color",
		"anim_effect_combo_stack",
	}
}

func animatedKeyframes() []animatedKeyframe {
	return []animatedKeyframe{
		{name: "first"},
		{name: "mid"},
		{name: "last"},
	}
}

func (k animatedKeyframe) frameIndex(totalFrames int) int {
	if totalFrames <= 1 {
		return 0
	}

	switch k.name {
	case "first":
		return 0
	case "mid":
		return totalFrames / 2
	case "last":
		return totalFrames - 1
	default:
		return 0
	}
}
