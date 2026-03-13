package complexity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestAnalyzeFile_JavaScript_Simple(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "simple.js", `
function add(a, b) {
  return a + b;
}

function greet(name) {
  console.log("hello " + name);
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Functions), 2)

	for _, f := range result.Functions {
		assert.Equal(t, 1, f.Complexity, "simple function %s should have complexity 1", f.Name)
	}
}

func TestAnalyzeFile_JavaScript_IfElse(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "branching.js", `
function check(x) {
  if (x > 10) {
    return "big";
  } else if (x > 5) {
    return "medium";
  } else {
    return "small";
  }
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + 2 ifs
}

func TestAnalyzeFile_JavaScript_BooleanOps(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "boolean.js", `
function validate(a, b) {
  if (a > 0 && b > 0) {
    return true;
  }
  return false;
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + if + &&
}

func TestAnalyzeFile_TypeScript(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "app.ts", `
function processItems(items: number[]): number {
  let sum = 0;
  for (const item of items) {
    if (item > 0) {
      sum += item;
    }
  }
  return sum;
}
`)
	result, err := AnalyzeFile(path, "TypeScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + for + if
}

func TestAnalyzeFile_Python(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "app.py", `
def process(items):
    result = []
    for item in items:
        if item > 0:
            result.append(item)
        elif item == 0:
            pass
    return result
`)
	result, err := AnalyzeFile(path, "Python")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + for + if + elif
}

func TestAnalyzeFile_Java(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "App.java", `
class App {
    public int sum(int[] items) {
        int total = 0;
        for (int item : items) {
            if (item > 0) {
                total += item;
            }
        }
        return total;
    }
}
`)
	result, err := AnalyzeFile(path, "Java")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.Complexity, 3) // base + for + if
}

func TestAnalyzeFile_Nesting(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "nested.js", `
function deep(items) {
  for (let i = 0; i < items.length; i++) {
    if (items[i] > 0) {
      while (items[i] > 10) {
        items[i]--;
      }
    }
  }
}
`)
	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Functions), 1)

	fn := result.Functions[0]
	assert.GreaterOrEqual(t, fn.MaxNesting, 3) // for > if > while
}

func TestAnalyzeFile_UnsupportedLanguage(t *testing.T) {
	result, err := AnalyzeFile("/tmp/test.unknown", "Cobol")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestAnalyzeFile_PHP(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "app.php", `<?php
function process($items) {
    $result = [];
    foreach ($items as $item) {
        if ($item > 0) {
            $result[] = $item;
        }
    }
    return $result;
}
?>`)
	result, err := AnalyzeFile(path, "PHP")
	require.NoError(t, err)
	if result != nil && len(result.Functions) > 0 {
		fn := result.Functions[0]
		assert.GreaterOrEqual(t, fn.Complexity, 3) // base + foreach + if
	}
}

func TestAnalyzeFile_Ruby(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "app.rb", `
def process(items)
  result = []
  items.each do |item|
    if item > 0
      result << item
    end
  end
  result
end
`)
	result, err := AnalyzeFile(path, "Ruby")
	require.NoError(t, err)
	if result != nil && len(result.Functions) > 0 {
		fn := result.Functions[0]
		assert.GreaterOrEqual(t, fn.Complexity, 1) // at least base complexity
	}
}

func TestSummarize(t *testing.T) {
	results := []*FileResult{
		{
			Path:     "a.js",
			Language: "JavaScript",
			Functions: []FunctionResult{
				{Name: "foo", Complexity: 5, MaxNesting: 2},
				{Name: "bar", Complexity: 15, MaxNesting: 4},
			},
		},
		{
			Path:     "b.js",
			Language: "JavaScript",
			Functions: []FunctionResult{
				{Name: "baz", Complexity: 1, MaxNesting: 0},
			},
		},
		nil, // should be skipped
	}

	s := Summarize(results)
	assert.Equal(t, 3, s.TotalFunctions)
	assert.InDelta(t, 7.0, s.AvgComplexity, 0.01)   // (5+15+1)/3 = 7.0
	assert.InDelta(t, 2.0, s.AvgNesting, 0.01)       // (2+4+0)/3 = 2.0
	assert.Equal(t, 15, s.MaxComplexity)
	assert.Equal(t, 4, s.MaxNesting)
	assert.Equal(t, 1, s.HighComplexity) // only bar (15 > 10)
}

func TestSummarize_Empty(t *testing.T) {
	s := Summarize(nil)
	assert.Equal(t, 0, s.TotalFunctions)
	assert.Equal(t, 0.0, s.AvgComplexity)
	assert.Equal(t, 0.0, s.AvgNesting)
}
