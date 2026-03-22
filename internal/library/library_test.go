package library

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func notFoundStat(string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func containsCandidatePath(candidates []libraryCandidate, want string) bool {
	for _, candidate := range candidates {
		if candidate.path == want {
			return true
		}
	}
	return false
}

func containsCandidateWithSource(candidates []libraryCandidate, path string, source string) bool {
	for _, candidate := range candidates {
		if candidate.path == path && candidate.source == source {
			return true
		}
	}
	return false
}

func normalizeForHost(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(path, "/", `\`)
	}
	return strings.ReplaceAll(path, `\`, "/")
}

func TestResolveLibraryLoadPlanRequiresPath(t *testing.T) {
	_, err := resolveLibraryLoadPlan("", runtime.GOOS, func(string) string { return "" }, notFoundStat)
	if err == nil {
		t.Fatal("expected an error when no explicit path or CHROMA_LIB_PATH is provided")
	}

	msg := err.Error()
	if !strings.Contains(msg, envLibPath) {
		t.Fatalf("expected missing-path error to mention %s, got: %q", envLibPath, msg)
	}
	if !strings.Contains(msg, defaultLibraryFilename(runtime.GOOS)) {
		t.Fatalf("expected missing-path error to mention expected filename, got: %q", msg)
	}
}

func TestResolveLibraryLoadPlanUsesInitPathBeforeEnv(t *testing.T) {
	plan, err := resolveLibraryLoadPlan(
		"./shim/target/debug/chroma_shim",
		runtime.GOOS,
		func(string) string { return "/ignored/from/env" },
		notFoundStat,
	)
	if err != nil {
		t.Fatalf("resolveLibraryLoadPlan returned error: %v", err)
	}

	if plan.configSource != "Init(libPath)" {
		t.Fatalf("expected Init path source, got: %s", plan.configSource)
	}

	expectedBase := normalizePathSeparators("./shim/target/debug/chroma_shim", runtime.GOOS)
	if !containsCandidateWithSource(plan.candidates, expectedBase, "configured") {
		t.Fatalf("expected candidates to include configured path %q, got: %#v", expectedBase, plan.candidates)
	}
}

func TestResolveLibraryLoadPlanUsesEnvWhenInitPathEmpty(t *testing.T) {
	plan, err := resolveLibraryLoadPlan(
		"",
		runtime.GOOS,
		func(key string) string {
			if key == envLibPath {
				return "./shim/target/debug/chroma_shim"
			}
			return ""
		},
		notFoundStat,
	)
	if err != nil {
		t.Fatalf("resolveLibraryLoadPlan returned error: %v", err)
	}

	if plan.configSource != envLibPath {
		t.Fatalf("expected env path source, got: %s", plan.configSource)
	}
}

func TestBuildLibraryPathCandidatesMissingExtension(t *testing.T) {
	tests := []struct {
		name  string
		goos  string
		input string
		want  []string
	}{
		{
			name:  "linux missing extension adds .so variants",
			goos:  "linux",
			input: "/tmp/chroma_shim",
			want:  []string{"/tmp/chroma_shim.so", "/tmp/libchroma_shim.so"},
		},
		{
			name:  "darwin missing extension adds .dylib variants",
			goos:  "darwin",
			input: "/tmp/chroma_shim",
			want:  []string{"/tmp/chroma_shim.dylib", "/tmp/libchroma_shim.dylib"},
		},
		{
			name:  "windows missing extension adds .dll only",
			goos:  "windows",
			input: `C:/tmp/chroma_shim`,
			want:  []string{`C:\tmp\chroma_shim`, `C:\tmp\chroma_shim.dll`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warnings := buildLibraryPathCandidates(tt.input, tt.goos, notFoundStat)
			if len(warnings) != 0 {
				t.Fatalf("expected no warnings for normal candidate generation, got: %#v", warnings)
			}
			for _, want := range tt.want {
				if !containsCandidatePath(got, want) {
					t.Fatalf("expected candidates to include %q, got: %#v", want, got)
				}
			}

			if tt.goos == "windows" && containsCandidatePath(got, `C:\tmp\libchroma_shim.dll`) {
				t.Fatalf("did not expect lib-prefixed Windows candidate, got: %#v", got)
			}
		})
	}
}

func TestBuildLibraryPathCandidatesWithExistingExtension(t *testing.T) {
	got, warnings := buildLibraryPathCandidates("/tmp/libchroma_shim.so", "linux", notFoundStat)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for existing-extension path, got: %#v", warnings)
	}
	if len(got) != 1 {
		t.Fatalf("expected one candidate for an already-qualified path, got %d (%#v)", len(got), got)
	}
	if got[0].source != "configured" {
		t.Fatalf("expected configured source for existing-extension path, got: %#v", got)
	}
}

func TestBuildLibraryPathCandidatesDirectoryInput(t *testing.T) {
	dir := t.TempDir()
	got, warnings := buildLibraryPathCandidates(dir, runtime.GOOS, os.Stat)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for directory path, got: %#v", warnings)
	}

	if len(got) != 1 {
		t.Fatalf("expected one directory-derived candidate, got %d (%#v)", len(got), got)
	}

	want := normalizePathSeparators(filepath.Join(dir, defaultLibraryFilename(runtime.GOOS)), runtime.GOOS)
	if got[0].path != want {
		t.Fatalf("expected candidate %q, got %q", want, got[0].path)
	}
	if got[0].source != "derived:directory-default-name" {
		t.Fatalf("expected directory-derived source, got: %#v", got[0])
	}
}

func TestLooksLikeDirectoryPathWithTrailingSeparator(t *testing.T) {
	isDir, warning := looksLikeDirectoryPath("/tmp/chroma/", nil)
	if warning != "" {
		t.Fatalf("expected no warning for trailing slash directory hint, got: %q", warning)
	}
	if !isDir {
		t.Fatal("expected trailing slash path to be treated as directory")
	}

	isDir, warning = looksLikeDirectoryPath(`C:\tmp\chroma\`, nil)
	if warning != "" {
		t.Fatalf("expected no warning for trailing backslash directory hint, got: %q", warning)
	}
	if !isDir {
		t.Fatal("expected trailing backslash path to be treated as directory")
	}
}

func TestLooksLikeDirectoryPathTrailingSeparatorWarnsWhenStatFails(t *testing.T) {
	isDir, warning := looksLikeDirectoryPath("/tmp/chroma/", notFoundStat)
	if !isDir {
		t.Fatal("expected trailing separator path to remain directory-intent even on stat failure")
	}
	if !strings.Contains(warning, "stat failed") {
		t.Fatalf("expected trailing-separator stat failure warning, got: %q", warning)
	}
}

func TestLooksLikeDirectoryPathNilStatWithoutTrailingSeparator(t *testing.T) {
	isDir, warning := looksLikeDirectoryPath("/tmp/chroma", nil)
	if isDir {
		t.Fatal("expected non-trailing path with nil stat to not infer directory")
	}
	if warning != "" {
		t.Fatalf("expected no warning with nil stat and non-trailing path, got: %q", warning)
	}
}

func TestResolveLibraryLoadPlanAddsAbsoluteFallbackForRelativePaths(t *testing.T) {
	relative := filepath.Join("shim", "target", "debug", defaultLibraryFilename(runtime.GOOS))
	relative = normalizePathSeparators(relative, runtime.GOOS)

	plan, err := resolveLibraryLoadPlan(relative, runtime.GOOS, nil, notFoundStat)
	if err != nil {
		t.Fatalf("resolveLibraryLoadPlan returned error: %v", err)
	}

	absolute, err := filepath.Abs(relative)
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}
	absolute = normalizeForHost(absolute)

	if !containsCandidateWithSource(plan.candidates, relative, "configured") {
		t.Fatalf("expected candidates to include configured path %q, got %#v", relative, plan.candidates)
	}
	if !containsCandidateWithSource(plan.candidates, absolute, "derived:absolute-fallback") {
		t.Fatalf("expected candidates to include absolute fallback path %q, got %#v", absolute, plan.candidates)
	}
}

func TestResolveLibraryLoadPlanSkipsAbsoluteFallbackForBareFilename(t *testing.T) {
	plan, err := resolveLibraryLoadPlan("libchroma_shim.so", "linux", nil, notFoundStat)
	if err != nil {
		t.Fatalf("resolveLibraryLoadPlan returned error: %v", err)
	}

	for _, candidate := range plan.candidates {
		if candidate.source == "derived:absolute-fallback" {
			t.Fatalf("did not expect absolute fallback for bare filename, got candidates %#v", plan.candidates)
		}
	}
}

func TestResolveLibraryLoadPlanIncludesAbsoluteFallbackWarnings(t *testing.T) {
	plan, err := resolveLibraryLoadPlanWithAbs(
		"./shim/target/debug/libchroma_shim",
		"linux",
		nil,
		notFoundStat,
		func(string) (string, error) {
			return "", errors.New("cwd unavailable")
		},
	)
	if err != nil {
		t.Fatalf("resolveLibraryLoadPlanWithAbs returned error: %v", err)
	}

	if len(plan.warnings) == 0 {
		t.Fatal("expected warning when absolute fallback resolution fails")
	}
	if !strings.Contains(plan.warnings[0], "cwd unavailable") {
		t.Fatalf("expected warning to include abs error, got: %#v", plan.warnings)
	}
}

func TestAppendAbsolutePathCandidatesWarnsWhenAbsResolverNil(t *testing.T) {
	base := []libraryCandidate{
		{path: "./shim/target/debug/libchroma_shim", source: "configured"},
	}

	got, warnings := appendAbsolutePathCandidates(base, "linux", nil)
	if len(got) != 1 {
		t.Fatalf("expected original candidate to remain, got %#v", got)
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning when abs resolver is nil")
	}
	if !strings.Contains(warnings[0], "abs resolver unavailable") {
		t.Fatalf("expected abs-resolver warning, got %#v", warnings)
	}
}

func TestResolveLibraryLoadPlanIncludesDirectoryStatWarnings(t *testing.T) {
	plan, err := resolveLibraryLoadPlan(
		"./restricted/chroma_shim",
		"linux",
		nil,
		func(string) (os.FileInfo, error) {
			return nil, os.ErrPermission
		},
	)
	if err != nil {
		t.Fatalf("resolveLibraryLoadPlan returned error: %v", err)
	}

	if len(plan.warnings) == 0 {
		t.Fatal("expected directory stat warning to be captured")
	}
	if !strings.Contains(plan.warnings[0], "ErrPermission") && !strings.Contains(plan.warnings[0], "permission") {
		t.Fatalf("expected warning to mention permission issue, got: %#v", plan.warnings)
	}
}

func TestPathExtForOSEdgeCases(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "libchroma_shim.so", want: ".so"},
		{path: "/tmp/.so", want: ""},
		{path: "/tmp/chroma.", want: ""},
		{path: "/tmp/noext", want: ""},
		{path: `C:\tmp\name.DLL`, want: ".dll"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := pathExtForOS(tt.path)
			if got != tt.want {
				t.Fatalf("pathExtForOS(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveConfiguredLibraryPathWithNilGetenv(t *testing.T) {
	path, source := resolveConfiguredLibraryPath("", nil)
	if path != "" || source != "" {
		t.Fatalf("expected empty path/source when getenv is nil and no explicit path, got (%q, %q)", path, source)
	}

	path, source = resolveConfiguredLibraryPath(" ./shim/target/debug/libchroma_shim.so ", nil)
	if path != "./shim/target/debug/libchroma_shim.so" {
		t.Fatalf("expected trimmed explicit path, got %q", path)
	}
	if source != "Init(libPath)" {
		t.Fatalf("expected Init source for explicit path, got %q", source)
	}
}

func TestIsAbsolutePathForOSTable(t *testing.T) {
	tests := []struct {
		name string
		goos string
		path string
		want bool
	}{
		{name: "unix absolute", goos: "linux", path: "/tmp/lib.so", want: true},
		{name: "unix relative", goos: "linux", path: "tmp/lib.so", want: false},
		{name: "windows drive absolute", goos: "windows", path: `C:\tmp\shim.dll`, want: true},
		{name: "windows drive relative", goos: "windows", path: `C:tmp\shim.dll`, want: false},
		{name: "windows unc absolute", goos: "windows", path: `\\server\share\shim.dll`, want: true},
		{name: "windows relative", goos: "windows", path: `tmp\shim.dll`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAbsolutePathForOS(tt.path, tt.goos)
			if got != tt.want {
				t.Fatalf("isAbsolutePathForOS(%q, %q) = %v, want %v", tt.path, tt.goos, got, tt.want)
			}
		})
	}
}

func TestFormatLoadAttempt(t *testing.T) {
	candidate := libraryCandidate{path: "/tmp/libchroma_shim.so", source: "configured"}

	msgWithErr := formatLoadAttempt(candidate, errors.New("open failed"))
	if !strings.Contains(msgWithErr, "[configured]") || !strings.Contains(msgWithErr, "open failed") {
		t.Fatalf("expected formatted load attempt to include source and error, got: %q", msgWithErr)
	}

	msgNilHandle := formatLoadAttempt(candidate, nil)
	if !strings.Contains(msgNilHandle, "unexpected") {
		t.Fatalf("expected nil-handle message to include severity hint, got: %q", msgNilHandle)
	}
}

func TestFormatLibraryLoadErrorIncludesCandidateKindsAndWarnings(t *testing.T) {
	plan := libraryLoadPlan{
		goos:         "linux",
		configured:   "./shim/target/debug/libchroma_shim",
		configSource: "Init(libPath)",
		candidates: []libraryCandidate{
			{path: "./shim/target/debug/libchroma_shim", source: "configured"},
			{path: "./shim/target/debug/libchroma_shim.so", source: "derived:added-extension"},
		},
		warnings: []string{"skipped absolute fallback for [configured] \"./shim/target/debug/libchroma_shim\": cwd unavailable"},
	}

	err := formatLibraryLoadError(plan, []string{
		"[configured] ./shim/target/debug/libchroma_shim (not found)",
		"[derived:added-extension] ./shim/target/debug/libchroma_shim.so (not found)",
	})
	msg := err.Error()

	if !strings.Contains(msg, "[configured] ./shim/target/debug/libchroma_shim") {
		t.Fatalf("expected error to include configured candidate marker, got: %q", msg)
	}
	if !strings.Contains(msg, "[derived:added-extension]") {
		t.Fatalf("expected error to include derived candidate marker, got: %q", msg)
	}
	if !strings.Contains(msg, "path resolution warnings") {
		t.Fatalf("expected error to include warnings section, got: %q", msg)
	}
}

func TestCandidateSetDeduplicatesAndSkipsEmptyValues(t *testing.T) {
	set := newCandidateSet(4)
	set.add("", "configured")
	set.add("   ", "configured")
	set.add("/tmp/libchroma_shim.so", "configured")
	set.add("/tmp/libchroma_shim.so", "derived:added-extension")

	got := set.candidates()
	if len(got) != 1 {
		t.Fatalf("expected one deduplicated candidate, got %#v", got)
	}
	if got[0].source != "configured" {
		t.Fatalf("expected first source to be retained on duplicate add, got %#v", got[0])
	}
}

func TestBuildLibraryPathCandidatesLibPrefixedInputDoesNotDoublePrefix(t *testing.T) {
	got, warnings := buildLibraryPathCandidates("/tmp/libchroma_shim", "linux", notFoundStat)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for lib-prefixed input, got %#v", warnings)
	}
	if containsCandidatePath(got, "/tmp/liblibchroma_shim.so") {
		t.Fatalf("did not expect double-lib candidate, got %#v", got)
	}
}

func TestBuildLibraryPathCandidatesOrdering(t *testing.T) {
	got, warnings := buildLibraryPathCandidates("/tmp/chroma_shim", "linux", notFoundStat)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(got) < 3 {
		t.Fatalf("expected at least 3 candidates, got %#v", got)
	}

	wantOrder := []string{
		"/tmp/chroma_shim",
		"/tmp/chroma_shim.so",
		"/tmp/libchroma_shim.so",
	}
	for i, want := range wantOrder {
		if got[i].path != want {
			t.Fatalf("expected candidate[%d]=%q, got %#v", i, want, got)
		}
	}
}

func TestAppendAbsolutePathCandidatesOrdering(t *testing.T) {
	base := []libraryCandidate{
		{path: "shim/a", source: "configured"},
		{path: "shim/b.so", source: "derived:added-extension"},
	}
	abs := func(path string) (string, error) {
		return "/abs/" + path, nil
	}

	got, warnings := appendAbsolutePathCandidates(base, "linux", abs)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	wantOrder := []string{
		"shim/a",
		"shim/b.so",
		"/abs/shim/a",
		"/abs/shim/b.so",
	}
	if len(got) != len(wantOrder) {
		t.Fatalf("expected %d candidates, got %#v", len(wantOrder), got)
	}
	for i, want := range wantOrder {
		if got[i].path != want {
			t.Fatalf("expected candidate[%d]=%q, got %#v", i, want, got)
		}
	}
}

func TestJoinPathForOSHandlesRootDirectories(t *testing.T) {
	if got := joinPathForOS("/", "libchroma_shim.so", "linux"); got != "/libchroma_shim.so" {
		t.Fatalf("expected unix root join to preserve root, got %q", got)
	}
	if got := joinPathForOS(`\`, "chroma_shim.dll", "windows"); got != `\chroma_shim.dll` {
		t.Fatalf("expected windows root join to preserve root, got %q", got)
	}
}
